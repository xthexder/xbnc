package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

type IRCServer struct {
	client *IRCClient

	connected bool
	sock      *net.TCPConn
	read      chan *IRCMessage
	write     chan string

	host     string
	addr     *net.TCPAddr
	nick     string
	login    string
	ident    string
	channels map[string]*IRCChannel
}

type IRCChannel struct {
	name   string
	active bool
}

func CreateServer(client *IRCClient, host, port, nick, login, ident string) (*IRCServer, error) {
	read := make(chan *IRCMessage, 1000)
	write := make(chan string, 1000)
	channels := make(map[string]*IRCChannel)
	addr, err := net.ResolveTCPAddr("tcp4", host+":"+port)
	if err != nil {
		return nil, err
	}
	return &IRCServer{client, false, nil, read, write, host, addr, nick, login, ident, channels}, nil
}

func (srv *IRCServer) Connect() error {
	if srv.connected {
		return nil
	}
	sock, err := net.DialTCP("tcp4", nil, srv.addr)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(sock)
	writer := bufio.NewWriter(sock)
	srv.sock = sock
	srv.connected = true

	go func() {
		for srv.connected {
			str, err := reader.ReadString('\n')
			if err != nil {
				continue
			}

			msg := ParseMessage(str[0 : len(str)-2]) // Cut off the \r\n and parse
			fmt.Printf("reads: %s\n", msg.raw)
			srv.read <- msg
		}
	}()
	go func() {
		for srv.connected {
			str := <-srv.write

			_, err := writer.WriteString(str + "\r\n")
			fmt.Printf("writes: %s\n", str)
			if err != nil {
				continue
			}
			writer.Flush()
		}
	}()

	srv.write <- "NICK " + srv.nick
	srv.write <- "USER " + srv.login + " 0 * :XBNC"

	for {
		msg := <-srv.read
		if msg == nil {
			continue
		}

		if msg.command == "PING" {
			srv.write <- "PONG :" + msg.message
		} else if msg.replycode >= 1 && msg.replycode <= 5 {
			srv.client.write <- ":-!xbnc@xbnc PRIVMSG " + srv.client.hostToChannel(srv.host, "") + " :" + msg.message
			// Successful connect
			break
		} else if msg.replycode == 433 {
			fmt.Printf("Nick already in use: %s\n", srv.nick)
			srv.nick = srv.nick + "_"
			srv.write <- "NICK " + srv.nick
		} else if msg.replycode >= 400 && msg.replycode != 439 {
			srv.Close()
			return errors.New("Could not log into IRC server: " + msg.raw)
		}
	}

	go srv.handler()
	return nil
}

func (srv *IRCServer) handler() {
	for srv.connected {
		msg := <-srv.read
		if msg == nil {
			continue
		}

		if msg.command == "PING" {
			srv.write <- "PONG :" + msg.message
		} else if msg.command == "REPLY" {
			srv.handleReplyCode(msg)
		} else if msg.command == "JOIN" {
			if msg.source == srv.nick {
				tmp, exists := srv.channels[msg.param[0]]
				if exists {
					tmp.active = true
				} else {
					srv.channels[msg.message] = &IRCChannel{msg.message, true}
				}
				srv.client.joinChannel(srv.client.hostToChannel(srv.host, msg.message), true)
			} else {
				srv.client.write <- ":" + msg.fullsource + " JOIN :" + srv.client.hostToChannel(srv.host, msg.message)
			}
		} else if msg.command == "PART" {
			if msg.source == srv.nick {
				_, exists := srv.channels[msg.param[0]]
				if exists {
					delete(srv.channels, msg.param[0])
				}
				srv.client.partChannel(srv.client.hostToChannel(srv.host, msg.param[0]))
			} else {
				srv.client.write <- ":" + msg.fullsource + " PART " + srv.client.hostToChannel(srv.host, msg.param[0]) + " :" + msg.message
			}
		} else if msg.command == "KICK" {
			if msg.param[1] == srv.nick {
				channel, exists := srv.channels[msg.param[0]]
				if exists && channel.active {
					channel.active = false
					go func(srv *IRCServer, name string) {
						time.Sleep(3 * time.Second)
						if srv.connected {
							channel, exists := srv.channels[name]
							if exists && !channel.active {
								srv.write <- "JOIN " + channel.name
							}
						}
					}(srv, channel.name)
				}
				srv.client.kickChannel(srv.client.hostToChannel(srv.host, msg.param[0]), msg.message)
			} else {
				srv.client.write <- ":" + msg.fullsource + " KICK " + srv.client.hostToChannel(srv.host, msg.param[0]) + " " + msg.param[1] + " :" + msg.message
			}
		} else if msg.command == "QUIT" {
			srv.client.write <- ":" + msg.fullsource + " QUIT :" + msg.message
		} else if msg.command == "PRIVMSG" {
			name := msg.param[0]
			if name == srv.nick {
				name = msg.source
			}
			channel := srv.client.hostToChannel(srv.host, name)
			srv.client.joinChannel(channel, false)
			srv.client.write <- ":" + msg.fullsource + " PRIVMSG " + channel + " :" + msg.message
		} else if msg.command == "NOTICE" {
			srv.client.write <- msg.raw
			if len(srv.ident) > 0 && msg.source == "NickServ" && strings.HasPrefix(msg.message, "This nickname is registered and protected") {
				srv.write <- "NICKSERV IDENTIFY " + srv.ident
			}
		} else if msg.command == "MODE" {
			if msg.paramlen == 4 {
				srv.client.write <- ":" + msg.fullsource + " MODE " + srv.client.hostToChannel(srv.host, msg.param[0]) + " " + msg.param[1] + " " + msg.param[2] + " " + msg.param[3]
			} else if msg.paramlen == 3 {
				srv.client.write <- ":" + msg.fullsource + " MODE " + srv.client.hostToChannel(srv.host, msg.param[0]) + " " + msg.param[1] + " " + msg.param[2]
			} else if msg.paramlen == 2 {
				srv.client.write <- ":" + msg.fullsource + " MODE " + srv.client.hostToChannel(srv.host, msg.param[0]) + " " + msg.param[1]
			} else if msg.paramlen == 1 {
				srv.client.write <- ":" + msg.fullsource + " MODE " + msg.param[0] + " :" + msg.message
			} else {
				srv.client.write <- ":-!xbnc@xbnc PRIVMSG " + srv.client.hostToChannel(srv.host, "") + " :" + msg.raw
			}
		} else if msg.command == "NICK" {
			srv.client.write <- msg.raw
		} else if msg.command == "TOPIC" {
			srv.client.write <- ":" + msg.fullsource + " TOPIC " + srv.client.hostToChannel(srv.host, msg.param[0]) + " :" + msg.message
		} else if msg.command == "CTCP_VERSION" {
			srv.client.write <- ":" + msg.source + "!xbnc@xbnc PRIVMSG " + srv.client.hostToChannel(srv.host, "") + " :Received CTCP VERSION: " + msg.raw
			srv.write <- "NOTICE " + msg.source + " :\x01XBNC 1.0: Created By xthexder\x01"
		} else {
			srv.client.write <- ":-!xbnc@xbnc PRIVMSG " + srv.client.hostToChannel(srv.host, "") + " :" + msg.raw
		}
	}
}

func (srv *IRCServer) handleReplyCode(msg *IRCMessage) {
	replycode := fmt.Sprintf("%03d", msg.replycode)
	if msg.replycode >= 1 && msg.replycode <= 3 {
		srv.client.write <- ":-!xbnc@xbnc PRIVMSG " + srv.client.hostToChannel(srv.host, "") + " :" + msg.message
	} else if (msg.replycode >= 4 && msg.replycode <= 5) || (msg.replycode >= 251 && msg.replycode <= 255) { // Server info
		tmpi := strings.Index(msg.raw, msg.param[0])
		if tmpi >= 0 && len(msg.param[0]) > 0 {
			tmpmsg := strings.TrimSpace(msg.raw[tmpi+len(msg.param[0]):])
			if strings.HasPrefix(tmpmsg, ":") {
				tmpmsg = tmpmsg[1:]
			}
			srv.client.write <- ":-!xbnc@xbnc PRIVMSG " + srv.client.hostToChannel(srv.host, "") + " :" + tmpmsg
		} else {
			srv.client.write <- ":-!xbnc@xbnc PRIVMSG " + srv.client.hostToChannel(srv.host, "") + " :" + msg.raw
		}
	} else if (msg.replycode >= 265 && msg.replycode <= 266) || msg.replycode == 375 || msg.replycode == 372 || msg.replycode == 376 { // Server info and MOTD
		srv.client.write <- ":-!xbnc@xbnc PRIVMSG " + srv.client.hostToChannel(srv.host, "") + " :" + msg.message
	} else if msg.replycode == 332 { // Channel topic
		srv.client.write <- ":" + conf.Hostname + " " + replycode + " " + msg.param[0] + " " + srv.client.hostToChannel(srv.host, msg.param[1]) + " :" + msg.message
	} else if msg.replycode == 333 { // Channel topic setter
		srv.client.write <- ":" + conf.Hostname + " " + replycode + " " + msg.param[0] + " " + srv.client.hostToChannel(srv.host, msg.param[1]) + " " + msg.param[2] + " " + msg.param[3]
	} else if msg.replycode == 353 { // Channel members
		srv.client.write <- ":" + conf.Hostname + " " + replycode + " " + msg.param[0] + " " + msg.param[1] + " " + srv.client.hostToChannel(srv.host, msg.param[2]) + " :" + msg.message
	} else if msg.replycode == 366 || msg.replycode == 315 { // Channel members/who end
		srv.client.write <- ":" + conf.Hostname + " " + replycode + " " + msg.param[0] + " " + srv.client.hostToChannel(srv.host, msg.param[1]) + " :" + msg.message
	} else if msg.replycode == 324 || msg.replycode == 329 { // Channel mode
		srv.client.write <- ":" + conf.Hostname + " " + replycode + " " + msg.param[0] + " " + srv.client.hostToChannel(srv.host, msg.param[1]) + " " + msg.param[2]
	} else if msg.replycode == 352 { // Channel who reply
		srv.client.write <- ":" + conf.Hostname + " " + replycode + " " + msg.param[0] + " " + srv.client.hostToChannel(srv.host, msg.param[1]) + " " + msg.param[2] + " " + msg.param[3] + " " + conf.Hostname + " " + msg.param[5] + " " + msg.param[6] + " :" + msg.message
	} else {
		srv.client.write <- ":-!xbnc@xbnc PRIVMSG " + srv.client.hostToChannel(srv.host, "") + " :" + msg.raw
	}
}

func (srv *IRCServer) Close() {
	srv.connected = false
	close(srv.read)
	close(srv.write)
	srv.channels = make(map[string]*IRCChannel)
	if srv.sock != nil {
		srv.sock.Close()
		srv.sock = nil
	}
}
