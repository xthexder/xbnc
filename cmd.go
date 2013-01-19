package main

import (
	"strconv"
	"strings"
)

func (client *IRCClient) handleXBNCCMD(msg string) {
	cmd := ParseMessage(msg)
	if cmd.command == "SERVER" {
		cmd2 := strings.ToUpper(cmd.param[0])
		if cmd2 == "ADD" {
			if len(cmd.param[1]) > 0 {
				client.addServer(strings.ToLower(cmd.param[1]), cmd.param[2])
			} else {
				client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :Usage: SERVER ADD <host> [ <port> ]"
			}
		} else if cmd2 == "REMOVE" {
			if len(cmd.param[1]) > 0 {
				client.removeServer(strings.ToLower(cmd.param[1]))
			} else {
				client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :Usage: SERVER REMOVE <host>"
			}
		} else {
			client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :Usage: SERVER <option> <parameters>"
			client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :Available options:"
			client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :   ADD      <host> [ <port> ]"
			client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :   REMOVE   <host>"
		}
	} else if cmd.command == "JOIN" || cmd.command == "PART" {
		if len(cmd.param[0]) > 0 && len(cmd.param[1]) > 0 {
			host := cmd.param[0]
			for server, id := range client.serverIds {
				if strconv.Itoa(id) == host {
					host = server
					break
				}
			}
			server := client.addServer(host, "6667")
			if server != nil {
				server.write <- cmd.command + " " + cmd.param[1]
			}
		} else {
			client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :Usage: " + cmd.command + " <host> <channel>"
		}
	} else {
		if cmd.command != "HELP" {
			client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :Command \"" + cmd.command + "\" not recognized"
		}
		client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :Available commands:"
		client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :   HELP      Display this help message"
		client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :   SERVER    Add or remove servers"
		client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :   JOIN      Join a channel on a server"
		client.write <- ":-!xbnc@xbnc PRIVMSG #xbnc :   PART      Part a channel on a server"
	}
}

func (srv *IRCServer) handleServerCMD(msg string) {
	serverchan := srv.client.hostToChannel(srv.host, "")
	cmd := ParseMessage(msg)
	if cmd.command == "JOIN" || cmd.command == "PART" {
		if len(cmd.param[0]) > 0 {
			srv.write <- cmd.command + " " + cmd.param[0]
		} else {
			srv.client.write <- ":-!xbnc@xbnc PRIVMSG " + serverchan + " :Usage: " + cmd.command + " <channel>"
		}
	} else if cmd.command == "CMD" {
		tmpcmd := strings.TrimSpace(cmd.raw[strings.Index(cmd.raw, "CMD")+4:])
		if len(tmpcmd) > 0 {
			srv.write <- tmpcmd
		} else {
			srv.client.write <- ":-!xbnc@xbnc PRIVMSG " + serverchan + " :Usage: CMD <command>"
		}
	} else {
		if cmd.command != "HELP" {
			srv.client.write <- ":-!xbnc@xbnc PRIVMSG " + serverchan + " :Command \"" + cmd.command + "\" not recognized"
		}
		srv.client.write <- ":-!xbnc@xbnc PRIVMSG " + serverchan + " :Available commands:"
		srv.client.write <- ":-!xbnc@xbnc PRIVMSG " + serverchan + " :   HELP      Display this help message"
		srv.client.write <- ":-!xbnc@xbnc PRIVMSG " + serverchan + " :   CMD       Send a raw command to this server"
		srv.client.write <- ":-!xbnc@xbnc PRIVMSG " + serverchan + " :   JOIN      Join a channel on this server"
		srv.client.write <- ":-!xbnc@xbnc PRIVMSG " + serverchan + " :   PART      Part a channel on this server"
	}
}
