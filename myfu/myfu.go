package myfu

import (
	"regexp"
	"os"
	"bufio"
	"fmt"
	"../github.com/alouca/gosnmp"
	"log"
	"sync"
	"strings"
	"os/exec"
)

const mib_percent_cpu_sys string = ".1.3.6.1.4.1.2021.11.9.0"
const mib_percent_cpu_usr string = ".1.3.6.1.4.1.2021.11.10.0"

func ExecCmd(cmd string, wg *sync.WaitGroup) {
	fmt.Println(cmd)
	parts := strings.Fields(cmd)
	out, err := exec.Command(parts[0],parts[1]).Output()
	if err != nil {
		fmt.Println("error occured")
		fmt.Printf("%s", err)
	}
	fmt.Printf("%s", out)
	wg.Done()
}

func GetCpuLoad(host string) int {
var sys,usr int

	s, err := gosnmp.NewGoSNMP(host, "public", gosnmp.Version2c, 5)
	if err != nil {
		log.Fatal(err)
	}
	cpu_sys, err := s.Get(mib_percent_cpu_sys)
	if err == nil {
		for _, v := range cpu_sys.Variables {
			switch v.Type {
			default:
				fmt.Printf("Type: %d - Value: %v\n", host, v.Value)
				sys = int(v.Value.(int))
			case gosnmp.OctetString:
				log.Printf("Response: %s : %s : %s \n", v.Name, v.Value.(string), v.Type.String())

			}
		}
	}
	cpu_usr, err := s.Get(mib_percent_cpu_usr)
	if err == nil {
		for _, v := range cpu_usr.Variables {
			switch v.Type {
			default:
				fmt.Printf("Type: %d - Value: %v\n", host, v.Value)
				usr = int(v.Value.(int))
			case gosnmp.OctetString:
			log.Printf("Response: %s : %s : %s \n", v.Name, v.Value.(string), v.Type.String())

			}
		}
	}


	return sys+usr
}

func Round(f float64) int {
	if f < -0.5 {
		return int(f - 0.5)
	}
	if f > 0.5 {
		return int(f + 0.5)
	}
	return 0
}

func Grep(re, filename string) string {
	regex, err := regexp.Compile(re)
	if err != nil {
		return "there was a problem with the regular expression"
	}

	fh, err := os.Open(filename)
	f := bufio.NewReader(fh)

	if err != nil {
		return "there was a problem opening the file"
	}
	defer fh.Close()

	buf := make([]byte, 1024)
	for {
		buf, _, err = f.ReadLine()
		if err != nil {
			return "error"
		}

		s := string(buf)
		if regex.MatchString(s) {
			//fmt.Printf("%s\n", string(buf))
			return s
		}
	}
}

func Grep2(re, filename string) {
	regex, err := regexp.Compile(re)
	if err != nil {
		return //"there was a problem with the regular expression"
	}

	fh, err := os.Open(filename)
	f := bufio.NewReader(fh)

	if err != nil {
		return //"there was a problem opening the file"
	}
	defer fh.Close()

	buf := make([]byte, 1024)
	for {
		buf, _, err = f.ReadLine()
		if err != nil {
			return //"error"
		}

		s := string(buf)
		if regex.MatchString(s) {
			fmt.Printf("%s\n", string(buf))
			//return s
		}
	}
}

func Replace(re, repl, filename string) {
	regex, err := regexp.Compile(re)
	if err != nil {
		return // there was a problem with the regular expression.
	}

	fh, err := os.Open(filename)
	f := bufio.NewReader(fh)

	if err != nil {
		return // there was a problem opening the file.
	}
	defer fh.Close()

	buf := make([]byte, 1024)
	for {
		buf, _ , err = f.ReadLine()
		if err != nil {
			return
		}

		s := string(buf)
		result := regex.ReplaceAllString(s, repl)
		fmt.Print(result + "\n")
	}
}

