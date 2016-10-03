package main

import (
	"fmt"
	"flag"
	//"./myfu"
	"github.com/alouca/gosnmp"
	"os"
	"encoding/json"
	"log"
	"bytes"
	"golang.org/x/crypto/ssh"
	//"./github.com/redacid/crypto/ssh"
	"github.com/fatih/color"
	"io/ioutil"
	"strings"
	"strconv"
	"net/smtp"


)

type Config struct {
	BackendServers  []BackendServer
	FrontendServers []FrontendServer
	GlobalConfigParams GlobalConfig
	//	ConfigGlobal ConfigGlobal
}

type BackendServer struct {
	Name          string `json:"name"`
	IP            string `json:"ip"`
	//Port       string `json:"port"`
	SSHPort       int `json:"sshPort"`
	Priority      int `json:"priority"`
	DefaultWeight int `json:"defaultWeight"`
	LastWeight    int `json:"lastWeight"`
	State         string `json:"state"`
}
type FrontendServer struct {
	Name          string `json:"name"`
	IP            string `json:"ip"`
	SSHPort       int `json:"sshPort"`
	NginxConfFile string `json:"NginxConfFile"`
}
type GlobalConfig struct {
	smtpHostPort          string `json:"smtpHostPort"`

}

//type ConfigGlobal struct {
//	RegExNginxServer string `json:"RegExNginxServer"`
//	NginxServerString string `json:"NginxServerString"`
//}

const mib_percent_cpu_sys string = ".1.3.6.1.4.1.2021.11.9.0"
const mib_percent_cpu_usr string = ".1.3.6.1.4.1.2021.11.10.0"

func GetCpuLoad(host string) int {
	var sys,usr int

	if strings.Contains(host,":") {
		LastDots := strings.LastIndex(host, ":")
		newhost := host[:LastDots]
		//fmt.Printf("!!!!!!!!! %s\n", newhost)
		host = newhost
	}

	//fmt.Printf("+++++++ %s\n", host)

	s, err := gosnmp.NewGoSNMP(host, "public", gosnmp.Version2c, 5)
	if err != nil {
		log.Fatal(err)
	}
	cpu_sys, err := s.Get(mib_percent_cpu_sys)
	if err == nil {
		for _, v := range cpu_sys.Variables {
			switch v.Type {
			default:
				//fmt.Printf("Type: %d - Value: %v\n", host, v.Value)
				sys = int(v.Value.(int))
			case gosnmp.OctetString:
			//log.Printf("Response: %s : %s : %s \n", v.Name, v.Value.(string), v.Type.String())

			}
		}
	}
	cpu_usr, err := s.Get(mib_percent_cpu_usr)
	if err == nil {
		for _, v := range cpu_usr.Variables {
			switch v.Type {
			default:
				//fmt.Printf("Type: %d - Value: %v\n", host, v.Value)
				usr = int(v.Value.(int))
			case gosnmp.OctetString:
			//log.Printf("Response: %s : %s : %s \n", v.Name, v.Value.(string), v.Type.String())

			}
		}
	}
	//fmt.Printf("Error: %v\n", err)

	return sys+usr
}

var command string
//var strForGrep string
//var strForRepl string
//var fileForGrep string
var writeWeightChanges string
var execCommand string
var srvName string
//var floatForRound float64


//var command = flag.String("command", "round", "Комманда(round...)")
//var floatForRound = flag.Float64("floatForRound",1.5, "Округлить до целого")
//var ip = flag.Int("flagname", 1234, "help message for flagname")

func executeCmd(cmd, hostname string, config *ssh.ClientConfig) string {
	conn, _ := ssh.Dial("tcp", hostname, config)
	session, _ := conn.NewSession()
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Run(cmd)

	//return hostname + ": " + stdoutBuf.String()
	return stdoutBuf.String()

}

func init() {
	//flag.StringVar(&command, "c", command, "Комманда(round,grep,replace,showconfig,changeweight ...)")
	flag.StringVar(&command, "command", command, "" +
		"Commands:\n " +
		"\t\t showConfig - Show configuration file \n" +
		"\t\t changeWeight - Change weight on Nginx frontends \n" +
		"\t\t snmpGetLoad - Get CPU load from backend servers via SNMP \n" +
		"\t\t getSrvStats - Get usage stats from server ( need -SrvName <name> ) and send it to e-mail \n" +
		"\t\t getStatsAll - Get usage stats from all servers and send it to e-mail \n" +
		"\t\t execOnBackends(need -execCommand <cmd>) - Execute command on backends \n "+
		"\t\t execOnFrontends(need -execCommand <cmd>) - Execute command on frontends \n ")

	flag.StringVar(&writeWeightChanges, "writeWeightChanges", writeWeightChanges, "(yes\\no) Write weight changes ( need by -command changeWeight) or only present changes\n")
	flag.StringVar(&execCommand, "execCommand", execCommand, "Exec command on servers(need by -command execOnFrontends or execOnBackends)\n")
	flag.StringVar(&srvName, "srvName", srvName, "Server name, hostname:sshport (need by -command getSrvStats)\n")

	//flag.Float64Var (&floatForRound, "round", floatForRound, "Число для округления до целого")
	//flag.StringVar (&strForGrep, "grep", strForGrep, "Строка(regex) для grep фильтра")
	//flag.StringVar (&strForRepl, "replace", strForRepl, "Строка для замены по grep фильтру")
	//flag.StringVar (&fileForGrep, "grepfile", strForGrep, "Файл для grep фильтра")
}

func main() {
	flag.Parse()

	f, err1 := os.OpenFile("/var/log/nginxweight.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err1 != nil {
		log.Fatalf("error opening log file: %v", err1)
	}
	defer f.Close()
	log.SetOutput(f)

	//Парсим файл конфигурации
	file, _ := os.Open("./config.json")
	decoder := json.NewDecoder(file)
	config := new(Config)
	err := decoder.Decode(&config)

	defer file.Close()

	if err != nil {
		//fmt.Printf("%s\n","Ошибка чтения файла конфигурации")
		log.Fatalf("Error read configuration file %v\n", err)

	}
	//Настройки SSH
	pkey, err := ioutil.ReadFile(os.Getenv("HOME") + "/.ssh/id_rsa")
	if err != nil {
		log.Fatalf("Can't open private key: %v", err)
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(pkey)
	if err != nil {
		log.Fatalf("Can't parse private key: %v", err)
	}

	sshConfig := &ssh.ClientConfig{
		User: os.Getenv("LOGNAME"),
		Auth: []ssh.AuthMethod{
			// Use the PublicKeys method for remote authentication.
			ssh.PublicKeys(signer),
		},
	}


	switch {
	default:
		//log.Fatal("Invalid or undefined command, type -h to help \n")
		fmt.Printf("%s", "Invalid or undefined command, type -h to help \n")

	case command == "showConfig":
		color.Red("Backend Servers")
		for _, BServer := range config.BackendServers {
			color.Cyan("Name: "+BServer.Name)
			fmt.Printf("IP: %s\n", BServer.IP)
			//fmt.Printf("Port: %d\n", BServer.Port)
			fmt.Printf("SSH Port: %d\n", BServer.SSHPort)
			fmt.Printf("State: %s\n", BServer.State)
			fmt.Printf("%s", "~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")
			//fmt.Printf("%s-%s:%d cpu_load:%d\n",BServer.Name,BServer.IP,BServer.SSHPort,myfu.GetCpuLoad(BServer.Name))
		}
		color.Green("Frontend Servers")
		for _, FServer := range config.FrontendServers {
			color.Cyan("Name: "+FServer.Name)
			fmt.Printf("IP: %s\n", FServer.IP)
			fmt.Printf("SSH Port: %d\n", FServer.SSHPort)
			fmt.Printf("Nginx config file: %s\n", FServer.NginxConfFile)
			fmt.Printf("%s", "~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")
		}
		fmt.Printf("smtpHostPort: %s\n",config.GlobalConfigParams.smtpHostPort)
	//fmt.Printf("%s\n",config.ConfigGlobal.NginxServerString)
	//fmt.Printf("%s\n",config.ConfigGlobal.RegExNginxServer)


	case command == "changeWeight":
		var BackendServerNewWeight int
		var BackendStateFlag string

		color.Green("Frontend Servers")
		for _, FServer := range config.FrontendServers {
			fmt.Printf("Server: %s-%s:%d (%s)\n", FServer.Name, FServer.IP, FServer.SSHPort, FServer.NginxConfFile)
			//color.Red("----------------------------------------------------")
			for _, BServer := range config.BackendServers {
				var sshcmd string


				//fmt.Printf("%s-%s:%d\n",BServer.Name,BServer.IP,BServer.SSHPort)
				fmt.Printf("%s(%s) %s cpu_load:%d\n", BServer.Name, BServer.IP, BServer.State, GetCpuLoad(BServer.Name))

				if BServer.State == "low" {
					BackendServerNewWeight = 1
				} else {
					//Нужно продумать формулу
					BackendServerNewWeight = 100 - GetCpuLoad(BServer.Name)
				}

				if BServer.State == "backup" {
					BackendStateFlag = "backup"

				} else if BServer.State == "down" {
					BackendStateFlag = "down"

				} else if BServer.State == "dynamic" {
					var BackendUpSeversCount int=0
					var BackendUpSeversSummaryLoad int=0

					//Нужно реализовать. Бэкэнд должен быть в бакапе пока нет нагрузки на остальные
					for _, BDServer := range config.BackendServers {
						if BDServer.State == "up" {
							BackendUpSeversCount = BackendUpSeversCount+1
							BackendUpSeversSummaryLoad = BackendUpSeversSummaryLoad+GetCpuLoad(BDServer.Name)
						}

					}
					//fmt.Printf("--- Up Servers(%d): %d\n",BackendUpSeversSummaryLoad, BackendUpSeversCount)
					AvgUpServersLoad := BackendUpSeversSummaryLoad/BackendUpSeversCount
					fmt.Printf("--- Up Servers(AvgLoad: %d) count: %d\n",AvgUpServersLoad, BackendUpSeversCount)
					if AvgUpServersLoad > 50 {
						BackendStateFlag = "up"

					} else {
						BackendStateFlag = "backup"
					}
					color.Cyan("New Dynamic state is "+BackendStateFlag)
					log.Println(FServer.Name+"-"+BServer.Name+" New Dynamic state is "+BackendStateFlag)

				} else {
					BackendStateFlag = ""

				}
				fmt.Printf("%s", "- Apply regexp to Nginx configuration file\n")
				NginxServerRegexp := "(server)(\\s+)(" + BServer.Name + ")(\\s+)(weight)(=)(\\d+)(\\s+)(max_fails)(=)(\\d+)(\\s+)(fail_timeout)(=)(5).*(;)"
				NginxServerLineCmd := "cat \"" + FServer.NginxConfFile + "\" | grep -P \"" + NginxServerRegexp + "\"| grep \"" + BServer.Name + "\" | sed 's/^[ \\t]*//' | grep -v ^\"#\" | head -n 1"
				NginxServerLine := executeCmd(NginxServerLineCmd, FServer.Name + ":" + strconv.Itoa(FServer.SSHPort), sshConfig)
				NginxServerNewLine := "server " + BServer.Name + " weight=" + strconv.Itoa(BackendServerNewWeight) + " max_fails=1 fail_timeout=5 " + BackendStateFlag + ";"

				if writeWeightChanges == "yes" {
					fmt.Printf("%s", "- Write changes to Nginx configuration file\n")
					sshcmd = "sed -i -e '/^[ \\t]*#/!s/" + strings.TrimRight(NginxServerLine, "\r\n") + "/" + NginxServerNewLine + "/g' " + FServer.NginxConfFile

				} else if writeWeightChanges == "no" {
					fmt.Printf("%s", "- Print changes in Nginx configuration file \n")
					sshcmd = "sed -e '/^[ \\t]*#/!s/" + strings.TrimRight(NginxServerLine, "\r\n") + "/" + NginxServerNewLine + "/g' " + FServer.NginxConfFile
				} else {
					log.Fatal("Param writeWeightChanges undefined, type -h to help")
					fmt.Print("Param writeWeightChanges undefined, type -h to help")
					os.Exit(1)
				}
				color.Cyan("New Weight is "+strconv.Itoa(BackendServerNewWeight))
				log.Println(FServer.Name+"-"+BServer.Name+" New Weight is "+strconv.Itoa(BackendServerNewWeight))
				fmt.Printf("%s\n", executeCmd(sshcmd, FServer.Name + ":" + strconv.Itoa(FServer.SSHPort), sshConfig))
				//fmt.Printf("%s", "~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")
			}
			if writeWeightChanges == "yes" {
				fmt.Printf("%s", "- Reload Nginx daemon \n\n")
				nginxReloadCmd := "/etc/init.d/nginx reload"
				fmt.Printf("%s\n", executeCmd(nginxReloadCmd, FServer.Name + ":" + strconv.Itoa(FServer.SSHPort), sshConfig))
			} else {
				fmt.Print("- Done.\n\n")
			}

		}
	case command == "snmpGetLoad":
		for _, BServer := range config.BackendServers {

			fmt.Printf("%s(%s) cpu_load:%d\n",BServer.Name,BServer.IP,GetCpuLoad(BServer.Name))
		}
	case command == "execOnFrontends":
		for _, FServer := range config.FrontendServers {
			execCmd := execCommand
			fmt.Printf("%s# %s\n",FServer.Name, executeCmd(execCmd, FServer.Name + ":" + strconv.Itoa(FServer.SSHPort), sshConfig))
		}

	case command == "execOnBackends":
		for _, BServer := range config.BackendServers {
			var host string
			execCmd := execCommand

			if strings.Contains(BServer.Name,":") {
				LastDots := strings.LastIndex(BServer.Name, ":")
				newhost := BServer.Name[:LastDots]
				//fmt.Printf("!!!!!!!!! %s\n", newhost)
				host = newhost
			} else {
				host = BServer.Name
			}


			//fmt.Printf("%s# %s %s %d\n",BServer.Name, execCmd, host, BServer.SSHPort)
			fmt.Printf("%s# %s\n",BServer.Name, executeCmd(execCmd, host + ":" + strconv.Itoa(BServer.SSHPort), sshConfig))
		}
	case command == "getStatsAll":
		var messagebody string
		execCmd := "top -b -n 1 | head -n 20 && iotop -b -n 1 -o"

		for _, FServer := range config.FrontendServers {
			messagebody = messagebody +"\n================ "+FServer.Name + "\n" + executeCmd(execCmd, FServer.Name + ":" + strconv.Itoa(FServer.SSHPort), sshConfig)
		}


		for _, BServer := range config.BackendServers {
			var host string
			if strings.Contains(BServer.Name,":") {
				LastDots := strings.LastIndex(BServer.Name, ":")
				newhost := BServer.Name[:LastDots]
				//fmt.Printf("!!!!!!!!! %s\n", newhost)
				host = newhost
			} else {
				host = BServer.Name
			}
			messagebody = messagebody +"\n================ "+ host + "\n" + executeCmd(execCmd, host + ":" + strconv.Itoa(BServer.SSHPort), sshConfig)
		}

		// Connect to the remote SMTP server.
		c, err := smtp.Dial("localhost:25")
		if err != nil {
			log.Fatal(err)
		}
		defer c.Close()
		// Set the sender and recipient.
		c.Mail("root")
		c.Rcpt("root")

		// Send the email body.
		wc, err := c.Data()
		if err != nil {
			log.Fatal(err)
		}
		defer wc.Close()
		buf := bytes.NewBufferString("Subject:All Servers Stats\n\n" + messagebody)
		if _, err = buf.WriteTo(wc); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s",messagebody)

	case command == "getSrvStats":
		var messagebody string
		execCmd := "top -b -n 1 | head -n 20 && iotop -b -n 1 -o"

		messagebody = messagebody +"\n================ "+ srvName + "\n" + executeCmd(execCmd, srvName, sshConfig)

		// Connect to the remote SMTP server.
		c, err := smtp.Dial("localhost:25")
		if err != nil {
			log.Fatal(err)
		}
		defer c.Close()
		// Set the sender and recipient.
		c.Mail("root")
		c.Rcpt("root")

		// Send the email body.
		wc, err := c.Data()
		if err != nil {
			log.Fatal(err)
		}
		defer wc.Close()
		buf := bytes.NewBufferString("Subject:"+srvName +" Server Stats\n\n" + messagebody)
		if _, err = buf.WriteTo(wc); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s",messagebody)
	/*
		case command == "round":
			fmt.Printf("%d", myfu.Round(floatForRound))

		case command == "grep":
			//fmt.Printf("%d", round(floatForRound))
			// ./goNginxWeightChanger -c grep -grep="(server)(\s+)(back4)(\s+)(weight)(=)(\d+)(\s+)(max_fails)(=)(\d+)(\s+)(fail_timeout)(=)(5)(;)" -grepfile="nginx.conf"
			myfu.Grep2(strForGrep, fileForGrep)

		case command == "replace":
			// ./goNginxWeightChanger -c replace -grep="(server)(\s+)(back4)(\s+)(weight)(=)(\d+)(\s+)(max_fails)(=)(\d+)(\s+)(fail_timeout)(=)(5)(;)" -replace "sdfsdfsdf" -grepfile="nginx.conf"
			myfu.Replace(strForGrep,strForRepl,fileForGrep)

	*/
	}
}