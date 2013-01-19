package main

import (
	"strconv"
	"strings"
	"time"
)

type IRCMessage struct {
	raw        string
	fullsource string
	source     string
	replycode  int
	command    string
	param      []string
	paramlen   int
	message    string
	time       int64
}

func tokenizeString(str string) (tokens []string, result string) {
	result = ""
	index := strings.Index(str, ":")
	if index >= 0 {
		result = str[index+1:]
		str = str[0:index]
	}
	tokens = strings.Fields(str)
	return
}

func ParseMessage(line string) *IRCMessage {
	msg := new(IRCMessage)
	msg.time = time.Now().Unix()
	msg.raw = line
	msg.command = "UNKNOWN"
	msg.param = make([]string, 100)
	msg.paramlen = 0
	if len(line) == 0 {
		return msg
	}
	colin := len(line) > 0 && line[0] == ':'
	if colin {
		line = line[1:]
	}
	tokens, message := tokenizeString(line)
	if len(tokens) > 0 {
		msg.message = message

		if len(tokens) == 1 {
			msg.command = strings.ToUpper(tokens[0])
		} else {
			if colin {
				msg.fullsource = tokens[0]
				tmp := strings.Index(tokens[0], "!")
				if tmp != -1 {
					msg.source = tokens[0][0:tmp]
				} else {
					msg.source = tokens[0]
				}

				msg.command = strings.ToUpper(tokens[1])

				msg.paramlen = len(tokens) - 2
				for i := 2; i < len(tokens) && i < 100; i++ {
					msg.param[i-2] = tokens[i]
				}
			} else {
				msg.command = strings.ToUpper(tokens[0])

				msg.paramlen = len(tokens) - 1
				for i := 1; i < len(tokens) && i < 100; i++ {
					msg.param[i-1] = tokens[i]
				}
			}

			code, err := strconv.Atoi(msg.command)
			if err == nil {
				msg.command = "REPLY"
				msg.replycode = code
			} else if msg.command == "PRIVMSG" {
				leng := len(message)
				if leng > 2 && message[0] == 1 && message[leng-1] == 1 {
					message = message[1 : leng-1]
					if strings.HasPrefix(message, "ACTION ") {
						msg.command = "PRIVMSG ACTION"
					} else if strings.HasPrefix(message, "DCC ") {
						msg.command = "PRIVMSG DCC"
					} else {
						switch message {
						case "PING":
							msg.command = "CTCP_PING"
						case "TIME":
							msg.command = "CTCP_TIME"
						case "VERSION":
							msg.command = "CTCP_VERSION"
						default:
							msg.command = "CTCP_OTHER"
						}
					}
				}
			}
		}
	}
	return msg
}
