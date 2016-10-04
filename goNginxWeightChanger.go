package main

import (
	"fmt"
	"flag"
	"github.com/alouca/gosnmp"
	"os"
	"encoding/json"
	"log"
	"bytes"
	"golang.org/x/crypto/ssh"
	"github.com/fatih/color"
	"io/ioutil"
	"strings"
	"strconv"
	"net/smtp"
	"path/filepath"
)

type Config struct {
	BackendServers  []BackendServer
	FrontendServers []FrontendServer
	Global
}

type BackendServer struct {
	Name          string `json:"name"`
	IP            string `json:"ip"`
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

type Global struct {
	SmtpHostPort string `json:"smtpHostPort"`
	LogFile string `json:"logFile"`
	NginxReloadCommand string `json:"nginxReloadCommand"`
	PercentDynamic int `json:"percentDynamic"`
	StatsCommand string `json:"statsCommand"`
	EmailFrom string `json:"emailFrom"`
	EmailTo string `json:"emailTo"`
}

const mib_percent_cpu_sys string = ".1.3.6.1.4.1.2021.11.9.0"
const mib_percent_cpu_usr string = ".1.3.6.1.4.1.2021.11.10.0"
//const mib_percent_cpu_idle string = ".1.3.6.1.4.1.2021.11.11.0"

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

	/*cpu_idle, err := s.Get(mib_percent_cpu_idle)
	if err == nil {
		for _, v := range cpu_idle.Variables {
			switch v.Type {
			default:
				//fmt.Printf("Type: %d - Value: %v\n", host, v.Value)
				idle = int(v.Value.(int))
			case gosnmp.OctetString:
			//log.Printf("Response: %s : %s : %s \n", v.Name, v.Value.(string), v.Type.String())

			}
		}
	}
	return idle*/
}

var command string
var writeWeightChanges string
var execCommand string
var srvName string


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

	flag.StringVar(&command, "command", command, "" +
		"Commands:\n " +
		"\t\t showConfig - Show configuration file \n" +
		"\t\t changeWeight - Change weight on Nginx frontends ( need -writeWeightChanges <yes\\no> )\n" +
		"\t\t snmpGetLoad - Get CPU load from backend servers via SNMP \n" +
		"\t\t getSrvStats - Get usage stats from server ( need -srvName <host:port> ) and send it to e-mail \n" +
		"\t\t getStatsAll - Get usage stats from all servers and send it to e-mail \n" +
		"\t\t execOnBackends(need -execCommand <cmd>) - Execute command on backends \n "+
		"\t\t execOnFrontends(need -execCommand <cmd>) - Execute command on frontends \n ")

	flag.StringVar(&writeWeightChanges, "writeWeightChanges", writeWeightChanges, "(yes\\no) Write weight changes ( need by -command changeWeight) or only present changes\n")
	flag.StringVar(&execCommand, "execCommand", execCommand, "Exec command on servers(need by -command execOnFrontends or execOnBackends)\n")
	flag.StringVar(&srvName, "srvName", srvName, "Server name, hostname:sshport (need by -command getSrvStats)\n")

}

func main() {
	flag.Parse()

	appdir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	//Парсим файл конфигурации
	file, _ := os.Open(appdir+"/config.json")
	decoder := json.NewDecoder(file)
	config := new(Config)
	err = decoder.Decode(&config)

	defer file.Close()

	f, err1 := os.OpenFile(config.LogFile, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err1 != nil {
		fmt.Printf("error opening log file: %v", err1)
		log.Fatalf("error opening log file: %v", err1)
	}
	defer f.Close()
	log.SetOutput(f)

	if err != nil {
		fmt.Printf("Error read configuration file %v\n", err)
		log.Fatalf("Error read configuration file %v\n", err)

	}
	//Настройки SSH
	pkey, err := ioutil.ReadFile(os.Getenv("HOME") + "/.ssh/id_rsa")
	if err != nil {
		fmt.Printf("Can't open private key: %v", err)
		log.Fatalf("Can't open private key: %v", err)
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(pkey)
	if err != nil {
		fmt.Printf("Can't parse private key: %v", err)
		log.Fatalf("Can't parse private key: %v", err)
	}

	sshConfig := &ssh.ClientConfig{
		User: os.Getenv("LOGNAME"),
		Auth: []ssh.AuthMethod{
			// Use the PublicKeys method for remote authentication.
			ssh.PublicKeys(signer),
		},
	}

	green := color.New(color.FgGreen).SprintFunc()
	//red := color.New(color.FgRed).SprintFunc()
	//yellow := color.New(color.FgYellow).SprintFunc()

	switch {
	default:
		//log.Fatal("Invalid or undefined command, type -h to help \n")
		fmt.Printf("%s", "Invalid or undefined command, type -h to help \n")
		log.Fatalf("%s", "Invalid or undefined command, type -h to help \n")

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
		fmt.Printf("smtpHostPort: %s\n",config.SmtpHostPort)
		fmt.Printf("logFile: %s\n",config.LogFile)
		fmt.Printf("NginxReloadCommand: %s\n",config.NginxReloadCommand)
		fmt.Printf("PercentDynamic: %d\n",config.PercentDynamic)
		fmt.Printf("StatsCommand: %s\n",config.StatsCommand)
		fmt.Printf("EmailFrom: %s\n",config.EmailFrom)
		fmt.Printf("EmailTo: %s\n",config.EmailTo)

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
				fmt.Printf("%s(%s) %s cpu_load:%d\n", BServer.Name, BServer.IP, green(BServer.State), GetCpuLoad(BServer.Name))

				if BServer.State == "low" {
					BackendServerNewWeight = 1
				} else {
					//Нужно продумать формулу
					BackendServerNewWeight = 100 - GetCpuLoad(BServer.Name)

						if BackendServerNewWeight <= 0 {
							BackendServerNewWeight = 1
						}
					//BackendServerNewWeight = GetCpuLoad(BServer.Name)
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
					if AvgUpServersLoad > config.PercentDynamic {
						//state UP
						BackendStateFlag = ""


					} else {
						BackendStateFlag = "backup"
					}
					color.Cyan("New Dynamic state is "+BackendStateFlag)
					log.Println(FServer.Name+"-"+BServer.Name+" New Dynamic state is "+BackendStateFlag)

				} else {
					//state UP
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
				//nginxReloadCmd := "/etc/init.d/nginx reload"
				fmt.Printf("%s\n", executeCmd(config.NginxReloadCommand, FServer.Name + ":" + strconv.Itoa(FServer.SSHPort), sshConfig))
			} else {
				fmt.Print("- Done.\n\n")
			}

		}
	case command == "snmpGetLoad":
		for _, BServer := range config.BackendServers {

			fmt.Printf("%s(%s) cpu_load:%d\n",BServer.Name,BServer.IP,GetCpuLoad(BServer.Name))
		}
	case command == "execOnFrontends":
		results := make(chan string)
		for _, FServer := range config.FrontendServers {

			execCmd := execCommand
			//fmt.Printf("%s# %s\n",FServer.Name, executeCmd(execCmd, FServer.Name + ":" + strconv.Itoa(FServer.SSHPort), sshConfig))




			go func() {
				results <- executeCmd(execCmd, FServer.Name + ":" + strconv.Itoa(FServer.SSHPort), sshConfig)
			}()
			res := <-results
			fmt.Print(res)

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
		//execCmd := "top -b -n 1 | head -n 20 && iotop -b -n 1 -o"
		execCmd := config.StatsCommand

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
		c, err := smtp.Dial(config.SmtpHostPort)
		if err != nil {
			log.Fatal(err)
		}
		defer c.Close()
		// Set the sender and recipient.
		c.Mail(config.EmailFrom)
		c.Rcpt(config.EmailTo)

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
		//execCmd := "top -b -n 1 | head -n 20 && iotop -b -n 1 -o"
		execCmd := config.StatsCommand
		messagebody = messagebody +"\n================ "+ srvName + "\n" + executeCmd(execCmd, srvName, sshConfig)

		// Connect to the remote SMTP server.
		c, err := smtp.Dial(config.SmtpHostPort)
		if err != nil {
			log.Fatal(err)
			fmt.Print(err)
		}
		defer c.Close()
		// Set the sender and recipient.
		c.Mail(config.EmailFrom)
		c.Rcpt(config.EmailTo)

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


	}
}
//https://go-tour-ua.appspot.com/concurrency/1