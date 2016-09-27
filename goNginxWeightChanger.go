package main

import (
	"fmt"
	"flag"
	"./myfu"
	"os"
	"encoding/json"
	"log"
	"bytes"
	"./github.com/redacid/crypto/ssh"
	"io/ioutil"
	"strings"
	"strconv"
)

type Config struct {
	BackendServers []BackendServer
	FrontendServers []FrontendServer
//	ConfigGlobal ConfigGlobal
}

type BackendServer struct {
	Name string `json:"name"`
	IP string `json:"ip"`
	SSHPort int `json:"sshPort"`
	Priority int `json:"priority"`
	DefaultWeight int `json:"defaultWeight"`
	LastWeight int `json:"lastWeight"`
	State string `json:"state"`
}
type FrontendServer struct {
	Name string `json:"name"`
	IP string `json:"ip"`
	SSHPort int `json:"sshPort"`
	NginxConfFile string `json:"NginxConfFile"`
}

//type ConfigGlobal struct {
//	RegExNginxServer string `json:"RegExNginxServer"`
//	NginxServerString string `json:"NginxServerString"`
//}


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
	flag.StringVar (&writeWeightChanges, "writeWeightChanges", writeWeightChanges, "(yes\\no) Записать извенения веса(changeweight), \n в  противном случае только показ изменений")
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
		log.Fatalf("%s\n","Ошибка чтения файла конфигурации")
	}

	switch {
	default:
		fmt.Printf("%s", "Не указана или неверная комманда введите -h для получения помощи\n")

	case command == "showconfig":
		fmt.Printf("%s","Backend Servers --------------------------------------\n")
		for _, BServer := range config.BackendServers {
			fmt.Printf("%s-%s:%d\n",BServer.Name,BServer.IP,BServer.SSHPort)
			//fmt.Printf("%s-%s:%d cpu_load:%d\n",BServer.Name,BServer.IP,BServer.SSHPort,myfu.GetCpuLoad(BServer.Name))
		}
		fmt.Printf("%s","Frontend Servers -------------------------------------\n")
		for _, FServer := range config.FrontendServers {
			fmt.Printf("%s-%s:%d (%s)\n",FServer.Name,FServer.IP,FServer.SSHPort,FServer.NginxConfFile)
		}
		//fmt.Printf("%s\n",config.ConfigGlobal.NginxServerString)
		//fmt.Printf("%s\n",config.ConfigGlobal.RegExNginxServer)


	case command == "changeweight":
		var BackendServerNewWeight int
		var BackendBackupFlag string

		fmt.Printf("%s","Frontend Servers -------------------------------------\n")
		for _, FServer := range config.FrontendServers {
			fmt.Printf("%s-%s:%d (%s)\n",FServer.Name,FServer.IP,FServer.SSHPort,FServer.NginxConfFile)

			pkey, err := ioutil.ReadFile(os.Getenv("HOME") + "/.ssh/id_rsa")
			if err != nil {
				log.Fatalf("unable to read private key: %v", err)
			}

			// Create the Signer for this private key.
			signer, err := ssh.ParsePrivateKey(pkey)
			if err != nil {
				log.Fatalf("unable to parse private key: %v", err)
			}

			sshConfig := &ssh.ClientConfig{
				User: os.Getenv("LOGNAME"),
				Auth: []ssh.AuthMethod{
					// Use the PublicKeys method for remote authentication.
					ssh.PublicKeys(signer),
				},
			}
			fmt.Printf("%s","Backend Servers --------------------------------------\n")
			for _, BServer := range config.BackendServers {
				var sshcmd string

				//fmt.Printf("%s-%s:%d\n",BServer.Name,BServer.IP,BServer.SSHPort)
				fmt.Printf("%s-%s:%d cpu_load:%d\n",BServer.Name,BServer.IP,BServer.SSHPort,myfu.GetCpuLoad(BServer.Name))

				if BServer.State == "disable" {
					BackendServerNewWeight = 1
				} else {
					BackendServerNewWeight = 100-myfu.GetCpuLoad(BServer.Name)
				}
				if BServer.State == "backup" {
					BackendBackupFlag = "backup"

				} else {
					BackendBackupFlag = ""

				}


				NginxServerRegexp := "(server)(\\s+)("+BServer.Name+")(\\s+)(weight)(=)(\\d+)(\\s+)(max_fails)(=)(\\d+)(\\s+)(fail_timeout)(=)(5).*(;)"
				NginxServerLineCmd := "cat \""+FServer.NginxConfFile+"\" | grep -P \""+NginxServerRegexp+"\"| grep \""+BServer.Name+"\" | sed 's/^[ \\t]*//' | grep -v ^\"#\" | head -n 1"
				NginxServerLine := executeCmd(NginxServerLineCmd, FServer.Name + ":" + strconv.Itoa(FServer.SSHPort), sshConfig)
				NginxServerNewLine := "server "+BServer.Name+" weight=" + strconv.Itoa(BackendServerNewWeight)+" max_fails=1 fail_timeout=5 "+BackendBackupFlag+"; #"+BServer.Name+""

				if writeWeightChanges == "yes" {
					sshcmd = "sed -i -e '/^[ \\t]*#/!s/"+ strings.TrimRight(NginxServerLine,"\r\n") +"/"+ NginxServerNewLine +"/g' "+FServer.NginxConfFile
				} else if writeWeightChanges == "no" {
					sshcmd = "sed -e '/^[ \\t]*#/!s/"+ strings.TrimRight(NginxServerLine,"\r\n") +"/"+ NginxServerNewLine +"/g' "+FServer.NginxConfFile
				} else {
					log.Fatalf("unable to parse private key: %v",nil)
					os.Exit(1)
				}

				fmt.Printf("%s\n",executeCmd(sshcmd, FServer.Name + ":" + strconv.Itoa(FServer.SSHPort), sshConfig))
			}
				nginxReloadCmd := "/etc/init.d/nginx reload"
			fmt.Printf("%s\n",executeCmd(nginxReloadCmd, FServer.Name + ":" + strconv.Itoa(FServer.SSHPort), sshConfig))

		}
	/*case command == "snmpget":
		for _, BServer := range config.BackendServers {

			fmt.Printf("%s-%s:%d cpu_load:%d\n",BServer.Name,BServer.IP,BServer.SSHPort,myfu.GetCpuLoad(BServer.Name))
		}

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