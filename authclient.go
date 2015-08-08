package main

import (
	"bufio"
	"fmt"
	"net"
)

type AuthClient struct {
	connected bool
	sock      *net.TCPConn
	read      chan *IRCMessage
	write     chan string
	client		*IRCClient
}

func AcceptAuthConnection(conn *net.TCPConn) {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	read := make(chan *IRCMessage, 1000)
	write := make(chan string, 1000)
	client := &AuthClient{true, conn, read, write, nil}

	go func() {
		for client.connected || (client.client != nil && client.client.connected) {
			str, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("readc error: %v\n", err)
				client.Close()
				break
			}

			msg := ParseMessage(str[0 : len(str)-2]) // Cut off the \r\n and parse
			if client.client != nil && client.client.connected {
				fmt.Printf("readc: %s\n", msg.raw)
				client.client.read <- msg
			} else {
				fmt.Printf("auth readc: %s\n", msg.raw)
				client.read <- msg
			}
		}
	}()
	go func() {
		for client.connected || (client.client != nil && client.client.connected) {
			var str string
			if client.client != nil && client.client.connected {
				str = <-client.client.write
				fmt.Printf("writec: %s\n", str)
			} else {
				str = <-client.write
				fmt.Printf("auth writec: %s\n", str)
			}
			if len(str) > 0 {
				_, err := writer.WriteString(str + "\r\n")
				if err != nil {
					fmt.Printf("writec error: %v\n", err)
					client.Close()
					break
				}
				writer.Flush()
			} else {
				if client.client != nil && !client.client.connected {
					client.client.sock = conn
					client.client.connected = true
					client.connected = false

					go client.client.Handler()
				} else {
					client.Close()
					break
				}
			}
		}
	}()
	go client.Handler()
}

func AttemptLogin(login, pass string) *IRCClient {
	if len(login) > 0 && len(pass) > 0 {
		client, exists := clients[login]
		if exists && client.pass == pass {
			return client
		}
	}
	return nil
}

func (client *AuthClient) SuccessfulAuth() {
	fmt.Printf("Client authenticated from %s\n", client.sock.RemoteAddr().String())

	client.write <- ":" + conf.Hostname + " 001 " + client.client.nick + " :Welcome to XBNC " + client.client.nick + "!" + client.client.login + "@xbnc"
	client.write <- ":" + conf.Hostname + " 002 " + client.client.nick + " :Your host is " + conf.Hostname + ", running version XBNC1.0"
	client.write <- ":" + conf.Hostname + " 003 " + client.client.nick + " :This server was created Tomorrow"
	client.write <- ":" + conf.Hostname + " 004 " + client.client.nick + " :" + conf.Hostname + " XBNC1.0 iowghraAsORTVSxNCWqBzvdHtGpI lvhopsmntikrRcaqOALQbSeIKVfMCuzNTGjZ"
	client.write <- ":" + conf.Hostname + " 005 " + client.client.nick + " :CHANTYPES=# NETWORK=XBNC PREFIX=(qaohv)~&@%+ CASEMAPPING=ascii :are supported by this server"

	client.write <- ":" + client.client.nick + "!" + client.client.login + "@xbnc JOIN :#xbnc"
	for host := range client.client.servers {
		client.write <- ":" + client.client.nick + "!" + client.client.login + "@xbnc JOIN :" + client.client.hostToChannel(host, "")
	}
	for name, channel := range client.client.channels {
		if channel.active {
			client.write <- ":" + client.client.nick + "!" + client.client.login + "@xbnc JOIN :" + name
		}
	}
	client.write <- ""
}

func (client *AuthClient) Handler() {
	var login, pass string

	for client.connected {
		msg := <-client.read
		switch msg.command {
			case "PING":
				client.write <- ":" + conf.Hostname + " PONG " + conf.Hostname + " :" + msg.param[0]
			case "USER":
				login = msg.param[0]
				ircclient := AttemptLogin(login, pass)
				if ircclient != nil {
					client.client = ircclient
					client.SuccessfulAuth()
					break
				} else {
					client.write <- ":" + conf.Hostname + " NOTICE AUTH :*** Please login with your password. /quote PASS <password>"
				}
			case "PASS":
				pass = msg.param[0]
				ircclient := AttemptLogin(login, pass)
				if ircclient != nil {
					client.client = ircclient
					client.SuccessfulAuth()
					break
				}
			case "QUIT":
				client.Close()
				break
		}
	}
}

func (client *AuthClient) Close() {
	client.connected = false
	client.write <- ""
	if client.client != nil {
		ircclient := client.client
		client.client = nil
		ircclient.connected = false
		ircclient.write <- ""
	}
	client.sock.Close()
	client.sock = nil
}
