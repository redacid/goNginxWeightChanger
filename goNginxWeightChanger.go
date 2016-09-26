package main

import (
	"fmt"
	"flag"
	//"regexp"
	//"os"
	//"bufio"
	"./myfu"
	"os"
	"encoding/json"
	"log"
	"os/exec"
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
}
type FrontendServer struct {
	Name string `json:"name"`
	IP string `json:"ip"`
	SSHPort int `json:"sshPort"`
	NginxConfigFile int `json:"NginxConfigFile"`
}

//type ConfigGlobal struct {
//	RegExNginxServer string `json:"RegExNginxServer"`
//	NginxServerString string `json:"NginxServerString"`
//}


var command string
var strForGrep string
var strForRepl string
var fileForGrep string
var floatForRound float64

//var command = flag.String("command", "round", "Комманда(round...)")
//var floatForRound = flag.Float64("floatForRound",1.5, "Округлить до целого")
//var ip = flag.Int("flagname", 1234, "help message for flagname")

func init() {
	flag.StringVar(&command, "c", command, "Комманда(round,grep,replace ...)")
	flag.Float64Var (&floatForRound, "round", floatForRound, "Число для округления до целого")
	flag.StringVar (&strForGrep, "grep", strForGrep, "Строка(regex) для grep фильтра")
	flag.StringVar (&strForRepl, "replace", strForRepl, "Строка для замены по grep фильтру")
	flag.StringVar (&fileForGrep, "grepfile", strForGrep, "Файл для grep фильтра")
}

func main() {
	flag.Parse()

	//Парсим файл конфигурации
	file, _ := os.Open("./config.json")
	decoder := json.NewDecoder(file)
	config := new(Config)
	err := decoder.Decode(&config)
	if err != nil {
		fmt.Printf("%s\n","Ошибка чтения файла конфигурации")
	}

	switch {
	default:
		fmt.Printf("%s", "Не указана или неверная комманда введите -h для получения помощи\n")
	case command == "showconfig":
		for _, BServer := range config.BackendServers {
			fmt.Printf("%s-%s:%d\n",BServer.Name,BServer.IP,BServer.SSHPort)
		}
		for _, FServer := range config.FrontendServers {
			fmt.Printf("%s-%s:%d (%s)\n",FServer.Name,FServer.IP,FServer.SSHPort,FServer.NginxConfigFile)
		}
		//fmt.Printf("%s\n",config.ConfigGlobal.NginxServerString)
		//fmt.Printf("%s\n",config.ConfigGlobal.RegExNginxServer)

		out, err := exec.Command("date").Output()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("The date is %s\n", out)

	case command == "round":
		fmt.Printf("%d", myfu.Round(floatForRound))

	case command == "grep":
		//fmt.Printf("%d", round(floatForRound))
		// ./goNginxWeightChanger -c grep -grep="(server)(\s+)(back4)(\s+)(weight)(=)(\d+)(\s+)(max_fails)(=)(\d+)(\s+)(fail_timeout)(=)(5)(;)" -grepfile="nginx.conf"
		myfu.Grep2(strForGrep, fileForGrep)

	case command == "replace":
		// ./goNginxWeightChanger -c replace -grep="(server)(\s+)(back4)(\s+)(weight)(=)(\d+)(\s+)(max_fails)(=)(\d+)(\s+)(fail_timeout)(=)(5)(;)" -replace "sdfsdfsdf" -grepfile="nginx.conf"
		myfu.Replace(strForGrep,strForRepl,fileForGrep)


	}
}