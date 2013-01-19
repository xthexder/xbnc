package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

var conf Config

type Config struct {
	Hostname string
	Nick     string
	Login    string
	Ident    string
}

func main() {
	file, err := ioutil.ReadFile("./config.json")
	if err != nil {
		fmt.Printf("Config file error: %v\n", err)
		return
	}
	err = json.Unmarshal(file, &conf)
	if err != nil {
		fmt.Printf("Config file error: %v\n", err)
		return
	}

	client := CreateClient(conf.Nick, conf.Login, conf.Ident)

	lisn, err := CreateListener(client, 1234)
	if err != nil {
		fmt.Println(err)
		return
	}

	/*reader := bufio.NewReader(os.Stdin)
	  for {
	    str, err := reader.ReadString('\n')
	    if err != nil {
	      fmt.Printf("stdin: %s\n", err)
	    }
	    srv.write <- "PRIVMSG #bnctest :" + str[0:len(str)-1]
	  }*/

	err = lisn.Listen()
	if err != nil {
		fmt.Println(err)
		return
	}
	<-make(chan int)
}
