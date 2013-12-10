package main

import (
	"encoding/json"
	"fmt"
	"github.com/howeyc/fsnotify"
	"io/ioutil"
	"os"
	"strings"
)

var (
	conf     Config
	listener *IRCListener
	clients  = make(map[string]*IRCClient)
)

type Config struct {
	Hostname string
	Port     int
}

type ClientConfig struct {
	Nick  string
	Login string
	Ident string
	Pass  string

	Servers map[string][]string
}

func readUserConfig(filename string) error {
	user := ClientConfig{}
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	err = json.Unmarshal(file, &user)
	if err != nil {
		return err
	}
	client, exists := clients[user.Login]
	if !exists {
		client = CreateClient(user.Nick, user.Login, user.Ident, user.Pass)
		clients[user.Login] = client
	}
	go func() {
		for hostport, channels := range user.Servers {
			tmp := strings.Split(hostport, ":")
			port := "6667"
			if len(tmp) > 1 {
				port = tmp[1]
			}
			server := client.addServer(tmp[0], port)
			for _, channel := range channels {
				_, exists := client.channels[client.hostToChannel(server.host, channel)]
				if !exists {
					server.write <- "JOIN " + channel
				}
			}
		}
	}()
	return nil
}

func watchHandler(watcher *fsnotify.Watcher) {
	for {
		select {
		case ev := <-watcher.Event:
			if ev != nil && ev.IsModify() {
				err := readUserConfig(ev.Name)
				if err != nil {
					fmt.Printf("User config error: %v\n", err)
				}
			}
		case err := <-watcher.Error:
			if err != nil {
				fmt.Printf("Config watcher error: %v\n", err)
			}
		}
	}
}

func main() {
	err := os.Mkdir("users", 0755)
	if err != nil && !os.IsExist(err) {
		fmt.Printf("Permissions error: %v\n", err)
		return
	}

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

	listener, err = CreateListener(conf.Hostname, conf.Port)
	if err != nil {
		fmt.Println(err)
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("Config watcher error: %v\n", err)
		return
	}
	defer watcher.Close()

	go watchHandler(watcher)

	err = watcher.Watch("users")
	if err != nil {
		fmt.Printf("Config watcher error: %v\n", err)
		return
	}

	files, err := ioutil.ReadDir("users")
	if err != nil {
		fmt.Printf("User config error: %v\n", err)
	}
	for _, file := range files {
		if !file.IsDir() {
			err := readUserConfig("users/" + file.Name())
			if err != nil {
				fmt.Printf("User config error: %v\n", err)
			}
		}
	}

	err = listener.Listen()
	if err != nil {
		fmt.Println(err)
	}
}
