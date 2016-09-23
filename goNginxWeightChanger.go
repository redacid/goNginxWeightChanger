package main

import (
	"fmt"
	"flag"
	//"regexp"
	//"os"
	//"bufio"
	"./myfu"

)

var command string
var strForGrep string
var fileForGrep string
var floatForRound float64

//var command = flag.String("command", "round", "Комманда(round...)")
//var floatForRound = flag.Float64("floatForRound",1.5, "Округлить до целого")
//var ip = flag.Int("flagname", 1234, "help message for flagname")

func init() {
	flag.StringVar(&command, "c", command, "Комманда(round,grep ...)")
	flag.Float64Var (&floatForRound, "round", floatForRound, "Число для округления до целого")
	flag.StringVar (&strForGrep, "grep", strForGrep, "Строка(regex) для grep фильтра")
	flag.StringVar (&fileForGrep, "grepfile", strForGrep, "Файл для grep фильтра")
}



//func main() {
//	flag.Parse()
//	if flag.NArg() == 3 {
//		repl(flag.Arg(0), flag.Arg(1), flag.Arg(2))
//	} else {
//		fmt.Printf("Wrong number of arguments.\n")
//	}
//}

func main() {
	flag.Parse()
	switch {
	default:
		fmt.Printf("%s", "Не указана или неверная комманда введите -h для получения помощи\n")
	case command == "round":
		fmt.Printf("%d", myfu.Round(floatForRound))
	case command == "grep":
		//fmt.Printf("%d", round(floatForRound))
		myfu.Grep2(strForGrep, fileForGrep)


	}
}