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
	//"os/exec"
	//"strconv"
	//"./github.com/alouca/gosnmp"
	//"sync"
	"bytes"
	"./github.com/evanphx/ssh"
	"io/ioutil"
	"io"
	"time"
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
	NginxConfFile string `json:"NginxConfFile"`
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

type SignerContainer struct {
	signers []ssh.Signer
}

func (t *SignerContainer) Key(i int) (key ssh.PublicKey, err error) {
	if i >= len(t.signers) {
		return
	}
	key = t.signers[i].PublicKey()
	return
}

func (t *SignerContainer) Sign(i int, rand io.Reader, data []byte) (sig []byte, err error) {
	if i >= len(t.signers) {
		return
	}
	sig, err = t.signers[i].Sign(rand, data)
	return
}

func makeSigner(keyname string) (signer ssh.Signer, err error) {
	fp, err := os.Open(keyname)
	if err != nil {
		return
	}
	defer fp.Close()

	buf, _ := ioutil.ReadAll(fp)
	signer, _ = ssh.ParsePrivateKey(buf)
	return
}

func makeKeyring() ssh.ClientAuth {
	signers := []ssh.Signer{}
	keys := []string{os.Getenv("HOME") + "/.ssh/id_rsa", os.Getenv("HOME") + "/.ssh/id_dsa"}

	for _, keyname := range keys {
		signer, err := makeSigner(keyname)
		if err == nil {
			signers = append(signers, signer)
		}
	}

	return ssh.ClientAuthKeyring(&SignerContainer{signers})
}

func executeCmd(cmd, hostname string, config *ssh.ClientConfig) string {
	conn, _ := ssh.Dial("tcp", hostname+":22", config)
	session, _ := conn.NewSession()
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Run(cmd)

	return hostname + ": " + stdoutBuf.String()
}


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
		}
		fmt.Printf("%s","Frontend Servers -------------------------------------\n")
		for _, FServer := range config.FrontendServers {
			fmt.Printf("%s-%s:%d (%s)\n",FServer.Name,FServer.IP,FServer.SSHPort,FServer.NginxConfFile)


			/*out, err := exec.Command("/usr/bin/ssh "+FServer.Name+" -p "+strconv.Itoa(FServer.SSHPort)+" date").Output()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("The date of "+FServer.Name+" is %s\n", out)

			wg := new(sync.WaitGroup)
			commands := []string{"/usr/bin/ssh "+FServer.Name+" -p "+strconv.Itoa(FServer.SSHPort)+" \"date\" "}
			for _, str := range commands {
				wg.Add(1)
				go myfu.ExecCmd(str, wg)
			}
			wg.Wait()*/

			cmd := "/usr/bin/whoami"
			host := FServer.Name

			results := make(chan string, 10)
			timeout := time.After(5 * time.Second)
			config := &ssh.ClientConfig{
				User: os.Getenv("LOGNAME"),
				Auth: []ssh.ClientAuth{makeKeyring()},
			}
			fmt.Printf("%s",executeCmd(cmd, host, config))


		}
		//fmt.Printf("%s\n",config.ConfigGlobal.NginxServerString)
		//fmt.Printf("%s\n",config.ConfigGlobal.RegExNginxServer)



	case command == "snmpget":
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


	}
}