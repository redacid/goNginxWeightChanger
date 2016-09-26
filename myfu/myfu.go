package myfu

import (
	"regexp"
	"os"
	"bufio"
	"fmt"
	"../github.com/alouca/gosnmp"
	"log"
)

const mib_percent_cpu_sys string = ".1.3.6.1.4.1.2021.11.9.0"
const mib_percent_cpu_usr string = ".1.3.6.1.4.1.2021.11.10.0"

func GetCpuLoad(host string) int {

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
				return v.Value
			case gosnmp.OctetString:
				//log.Printf("Response: %s : %s : %s \n", v.Name, v.Value.(string), v.Type.String())

			}
		}
	}



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

