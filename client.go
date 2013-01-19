package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type IRCClient struct {
	connected bool
	sock      *net.TCPConn
	read      chan *IRCMessage
	write     chan string

	channels map[string]*IRCChannel

	nick      string
	login     string
	ident     string
	servers   map[string]*IRCServer
	serverIds map[string]int
	nextId    chan int
}

func CreateClient(nick, login, ident string) *IRCClient {
	read := make(chan *IRCMessage, 1000)
	write := make(chan string, 1000)
	channels := make(map[string]*IRCChannel)
	servers := make(map[string]*IRCServer)
	serverIds := make(map[string]int)
	nextId := make(chan int, 100)
	for i := 1; i <= 100; i++ {
		nextId <- i
	}
	return &IRCClient{false, nil, read, write, channels, nick, login, ident, servers, serverIds, nextId}
}

func (client *IRCClient) handler() {
	fmt.Printf("Got connection from %s\n", client.sock.RemoteAddr().String())

	for client != nil && client.connected {
		msg := <-client.read

		if msg.command == "PING" {
			client.write <- ":" + conf.Hostname + " PONG " + conf.Hostname + " :" + msg.param[0]
		} else if msg.command == "USER" {
			client.login = msg.param[0]
			client.write <- ":" + conf.Hostname + " 001 " + client.nick + " :Welcome to XBNC " + client.nick + "!" + client.login + "@xbnc"
			client.write <- ":" + conf.Hostname + " 002 " + client.nick + " :Your host is " + conf.Hostname + ", running version XBNC1.0"
			client.write <- ":" + conf.Hostname + " 003 " + client.nick + " :This server was created Tomorrow"
			client.write <- ":" + conf.Hostname + " 004 " + client.nick + " :" + conf.Hostname + " XBNC1.0 iowghraAsORTVSxNCWqBzvdHtGpI lvhopsmntikrRcaqOALQbSeIKVfMCuzNTGjZ"
			client.write <- ":" + conf.Hostname + " 005 " + client.nick + " :CHANTYPES=# NETWORK=XBNC PREFIX=(qaohv)~&@%+ CASEMAPPING=ascii :are supported by this server"

			client.write <- ":" + client.nick + "!" + client.login + "@xbnc JOIN :#xbnc"
		} else if msg.command == "PRIVMSG" && msg.param[0] == "#xbnc" {
			client.handleXBNCCMD(msg.message)
		} else if msg.command == "JOIN" {
			host, channel := client.channelToHost(msg.param[0])
			if len(host) > 0 {
				server := client.addServer(host, msg.param[1])
				if server != nil {
					if len(channel) > 0 {
						server.write <- "JOIN " + channel
					} else {
						client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :Could not find channel to join: " + msg.param[0]
					}
				} else {
					client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :Could not find server to join: " + msg.param[0]
				}
			} else {
				client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :Could not find server to join: " + msg.param[0]
			}
		} else if msg.command == "PART" {
			if msg.param[0] == "#xbnc" {
				client.write <- ":" + client.nick + "!" + client.login + "@xbnc JOIN :#xbnc"
			} else {
				host, channel := client.channelToHost(msg.param[0])

				server, exists := client.servers[host]
				if exists {
					if len(channel) > 0 {
						ch, exists := client.channels[channel]
						if exists && ch.active {
							server.write <- "PART " + channel + " :Leaving"
						} else {
							client.partChannel(channel)
						}
					} else {
						client.removeServer(host)
					}
				} else {
					client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :Could not find server to part: " + msg.param[0]
				}
			}
		} else if msg.command == "PRIVMSG" {
			host, channel := client.channelToHost(msg.param[0])
			server, exists := client.servers[host]
			if exists && len(channel) > 0 {
				server.write <- "PRIVMSG " + channel + " :" + msg.message
			} else if exists {
				server.handleServerCMD(msg.message)
			}
		} else if msg.command == "MODE" {
			if strings.HasPrefix(msg.param[0], "#") {
				host, channel := client.channelToHost(msg.param[0])
				server, exists := client.servers[host]
				if exists && len(channel) > 0 {
					if len(msg.param[1]) > 0 {
						if len(msg.param[2]) > 0 {
							server.write <- "MODE " + channel + " " + msg.param[1] + " " + msg.param[2]
						} else {
							server.write <- "MODE " + channel + " " + msg.param[1]
						}
					} else {
						server.write <- "MODE " + channel
					}
				} else {
					client.write <- ":client!xbnc@xbnc PRIVMSG #xbnc :" + msg.raw
				}
			} else {
				client.write <- ":client!xbnc@xbnc PRIVMSG #xbnc :" + msg.raw
			}
		} else if msg.command == "WHO" {
			if strings.HasPrefix(msg.param[0], "#") {
				host, channel := client.channelToHost(msg.param[0])
				server, exists := client.servers[host]
				if exists && len(channel) > 0 {
					if len(msg.param[1]) > 0 {
						server.write <- "WHO " + channel + " " + msg.param[1]
					} else {
						server.write <- "WHO " + channel
					}
				} else {
					client.write <- ":client!xbnc@xbnc PRIVMSG #xbnc :" + msg.raw
				}
			} else {
				client.write <- ":client!xbnc@xbnc PRIVMSG #xbnc :" + msg.raw
			}
		} else {
			client.write <- ":client!xbnc@xbnc PRIVMSG #xbnc :" + msg.raw
		}
	}
}

func (client *IRCClient) joinChannel(name string, server bool) {
	channel, exists := client.channels[name]
	if exists {
		if channel.active {
			return
		} else if server {
			channel.active = true
		}
	}
	client.channels[name] = &IRCChannel{name, server}
	client.write <- ":" + client.nick + "!" + client.login + "@xbnc JOIN :" + name
}

func (client *IRCClient) partChannel(name string) {
	_, exists := client.channels[name]
	if exists {
		delete(client.channels, name)
		client.write <- ":" + client.nick + "!" + client.login + "@xbnc PART " + name + " :Leaving"
	}
}

func (client *IRCClient) kickChannel(name, reason string) {
	_, exists := client.channels[name]
	if exists {
		delete(client.channels, name)
		client.write <- ":" + client.nick + "!" + client.login + "@xbnc KICK " + name + " " + client.nick + " :" + reason
	}
}

func (client *IRCClient) addServer(host, port string) *IRCServer {
	server, exists := client.servers[host]
	if exists {
		return server
	}
	if len(port) == 0 {
		port = "6667"
	}
	srv, err := CreateServer(client, host, port, client.nick, client.login, "x8rx8r12")
	if err != nil {
		fmt.Println(err)
		client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :Error: " + err.Error()
		return nil
	}
	err = srv.Connect()
	if err != nil {
		fmt.Println(err)
		client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :Error: " + err.Error()
		return nil
	}
	client.servers[host] = srv
	client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :Connected to \"" + host + "\" successfully"
	client.write <- ":" + client.nick + "!" + client.login + "@xbnc JOIN :" + client.hostToChannel(host, "")
	return srv
}

func (client *IRCClient) removeServer(host string) {
	server, exists := client.servers[host]
	if exists {
		for _, channel := range server.channels {
			client.write <- ":" + client.nick + "!" + client.login + "@xbnc PART " + client.hostToChannel(server.host, channel.name) + " :Leaving"
		}
		server.Close()
		client.write <- ":" + client.nick + "!" + client.login + "@xbnc PART :" + client.hostToChannel(host, "")
		delete(client.servers, host)
		client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :Disconnected from \"" + host + "\" successfully"
	} else {
		client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :Could not remove \"" + host + "\", not found"
	}
}

func (client *IRCClient) channelToHost(channel string) (string, string) {
	host := channel
	param := ""
	if strings.HasPrefix(host, "#") {
		host = host[1:]
	}
	index := strings.Index(host, "-")
	if index >= 0 {
		param = host[index+1:]
		host = host[:index]
	}
	hostid, err := strconv.Atoi(host)
	if err == nil {
		if strings.HasPrefix(param, "!") {
			return param[1:], ""
		}
		for server, id := range client.serverIds {
			if id == hostid {
				return server, param
			}
		}
	}
	return host, param
}

func (client *IRCClient) hostToChannel(host, channel string) string {
	id, exists := client.serverIds[host]
	if !exists {
		id = <-client.nextId
		client.serverIds[host] = id
	}
	if len(channel) > 0 {
		return "#" + strconv.Itoa(id) + "-" + channel
	}
	return "#" + strconv.Itoa(id) + "-!" + host
}

func (client *IRCClient) Close() {
	client.connected = false
	close(client.read)
	close(client.write)
	client.sock.Close()
}
