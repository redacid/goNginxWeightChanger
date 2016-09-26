package myfu

import (
	"regexp"
	"os"
	"bufio"
	"fmt"
)



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

