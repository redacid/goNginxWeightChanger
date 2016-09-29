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


)

type Config struct {
	BackendServers  []BackendServer
	FrontendServers []FrontendServer
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
	flag.StringVar(&command, "c", command, "Комманда(showconfig,changeweight ...)")
	flag.StringVar(&writeWeightChanges, "writeWeightChanges", writeWeightChanges, "(yes\\no) Записать извенения веса(changeweight), \n в  противном случае только показ изменений")
	//flag.Float64Var (&floatForRound, "round", floatForRound, "Число для округления до целого")
	//flag.StringVar (&strForGrep, "grep", strForGrep, "Строка(regex) для grep фильтра")
	//flag.StringVar (&strForRepl, "replace", strForRepl, "Строка для замены по grep фильтру")
	//flag.StringVar (&fileForGrep, "grepfile", strForGrep, "Файл для grep фильтра")
}

func main() {
	flag.Parse()

	//Парсим файл конфигурации
	file, _ := os.Open("./config.json")
	decoder := json.NewDecoder(file)
	config := new(Config)
	err := decoder.Decode(&config)
	if err != nil {
		//fmt.Printf("%s\n","Ошибка чтения файла конфигурации")
		log.Fatalf("Ошибка чтения файла конфигурации %v\n", err)

	}

	switch {
	default:
		fmt.Printf("%s", "Не указана или неверная комманда введите -h для получения помощи\n")

	case command == "showconfig":
		color.Red("-------------------------------------- Backend Servers -------------------------------------")
		for _, BServer := range config.BackendServers {
			color.Cyan("Name: "+BServer.Name)
			fmt.Printf("IP: %s\n", BServer.IP)
			//fmt.Printf("Port: %d\n", BServer.Port)
			fmt.Printf("SSH Port: %d\n", BServer.SSHPort)
			fmt.Printf("State: %s\n", BServer.State)
			fmt.Printf("%s", "~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")
			//fmt.Printf("%s-%s:%d cpu_load:%d\n",BServer.Name,BServer.IP,BServer.SSHPort,myfu.GetCpuLoad(BServer.Name))
		}
		color.Green("-------------------------------------- Frontend Servers -------------------------------------")
		for _, FServer := range config.FrontendServers {
			color.Cyan("Name: "+FServer.Name)
			fmt.Printf("IP: %s\n", FServer.IP)
			fmt.Printf("SSH Port: %d\n", FServer.SSHPort)
			fmt.Printf("Nginx config file: %s\n", FServer.NginxConfFile)
			fmt.Printf("%s", "~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")
		}
	//fmt.Printf("%s\n",config.ConfigGlobal.NginxServerString)
	//fmt.Printf("%s\n",config.ConfigGlobal.RegExNginxServer)


	case command == "changeweight":
		var BackendServerNewWeight int
		var BackendStateFlag string

		color.Green("-------------------------------------- Frontend Servers -------------------------------------")
		for _, FServer := range config.FrontendServers {
			fmt.Printf("%s-%s:%d (%s)\n", FServer.Name, FServer.IP, FServer.SSHPort, FServer.NginxConfFile)

			pkey, err := ioutil.ReadFile(os.Getenv("HOME") + "/.ssh/id_rsa")
			if err != nil {
				log.Fatalf("Не могу прочитать приватный ключ: %v", err)
			}

			// Create the Signer for this private key.
			signer, err := ssh.ParsePrivateKey(pkey)
			if err != nil {
				log.Fatalf("Не могу распарсить приватный ключ: %v", err)
			}

			sshConfig := &ssh.ClientConfig{
				User: os.Getenv("LOGNAME"),
				Auth: []ssh.AuthMethod{
					// Use the PublicKeys method for remote authentication.
					ssh.PublicKeys(signer),
				},
			}
			color.Red("-------------------------------------- Backend Servers -------------------------------------")
			for _, BServer := range config.BackendServers {
				var sshcmd string

				//fmt.Printf("%s-%s:%d\n",BServer.Name,BServer.IP,BServer.SSHPort)
				fmt.Printf("%s(%s) cpu_load:%d\n", BServer.Name, BServer.IP, GetCpuLoad(BServer.Name))




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
					BackendStateFlag = ""
					//Нужно реализовать. Бэкэнд должен быть в бакапе пока нет нагрузки на остальные
				} else {
					BackendStateFlag = ""

				}
				fmt.Printf("%s", "- Применяем regexp к конфигу nginx\n")
				NginxServerRegexp := "(server)(\\s+)(" + BServer.Name + ")(\\s+)(weight)(=)(\\d+)(\\s+)(max_fails)(=)(\\d+)(\\s+)(fail_timeout)(=)(5).*(;)"
				NginxServerLineCmd := "cat \"" + FServer.NginxConfFile + "\" | grep -P \"" + NginxServerRegexp + "\"| grep \"" + BServer.Name + "\" | sed 's/^[ \\t]*//' | grep -v ^\"#\" | head -n 1"
				NginxServerLine := executeCmd(NginxServerLineCmd, FServer.Name + ":" + strconv.Itoa(FServer.SSHPort), sshConfig)
				NginxServerNewLine := "server " + BServer.Name + " weight=" + strconv.Itoa(BackendServerNewWeight) + " max_fails=1 fail_timeout=5 " + BackendStateFlag + ";"

				if writeWeightChanges == "yes" {
					fmt.Printf("%s", "-Производим замену в конфиге Nginx\n")
					sshcmd = "sed -i -e '/^[ \\t]*#/!s/" + strings.TrimRight(NginxServerLine, "\r\n") + "/" + NginxServerNewLine + "/g' " + FServer.NginxConfFile
				} else if writeWeightChanges == "no" {
					fmt.Printf("%s", "-Выводим изменения в конфиге Nginx\n")
					sshcmd = "sed -e '/^[ \\t]*#/!s/" + strings.TrimRight(NginxServerLine, "\r\n") + "/" + NginxServerNewLine + "/g' " + FServer.NginxConfFile
				} else {
					log.Fatal("Не определен параметр writeWeightChanges, введите -h для помощи")
					os.Exit(1)
				}

				fmt.Printf("%s\n", executeCmd(sshcmd, FServer.Name + ":" + strconv.Itoa(FServer.SSHPort), sshConfig))
				fmt.Printf("%s", "~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")
			}
			if writeWeightChanges == "yes" {
				fmt.Printf("%s", "- Релоадим Nginx\n")
				//nginxReloadCmd := "/etc/init.d/nginx reload"
				//fmt.Printf("%s\n", executeCmd(nginxReloadCmd, FServer.Name + ":" + strconv.Itoa(FServer.SSHPort), sshConfig))
			} else {
				fmt.Print("- Готово.\n")
			}

		}
	case command == "snmpget":
		for _, BServer := range config.BackendServers {

			fmt.Printf("%s(%s) cpu_load:%d\n",BServer.Name,BServer.IP,GetCpuLoad(BServer.Name))
		}
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