package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
)

type IRCListener struct {
	listening bool
	addr      *net.TCPAddr

	client *IRCClient
}

func CreateListener(client *IRCClient, port int) (*IRCListener, error) {
	addr, err := net.ResolveTCPAddr("tcp4", ":"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}
	return &IRCListener{false, addr, client}, nil
}

func (lisn *IRCListener) Listen() error {
	if lisn.client == nil {
		return errors.New("Client was not created!")
	}
	if lisn.client.sock != nil || lisn.listening {
		return nil
	}

	listener, err := net.ListenTCP("tcp4", lisn.addr)
	if err != nil {
		return err
	}

	lisn.listening = true

	fmt.Printf("Listening for TCP on %d\n", lisn.addr.Port)

	go func() {
		for lisn.listening {
			conn, err := listener.AcceptTCP()
			if err != nil {
				fmt.Printf("Listen error: %v\n", err)
				continue
			}

			lisn.client.sock = conn
			lisn.client.connected = true
			lisn.listening = false

			reader := bufio.NewReader(conn)
			writer := bufio.NewWriter(conn)

			go func() {
				for lisn.client != nil && lisn.client.connected {
					str, err := reader.ReadString('\n')
					if err != nil {
						fmt.Printf("readc error: %v\n", err)
						break
					}

					msg := ParseMessage(str[0 : len(str)-2]) // Cut off the \r\n and parse
					fmt.Printf("readc: %s\n", msg.raw)
					lisn.client.read <- msg
				}
			}()
			go func() {
				for lisn.client != nil && lisn.client.connected {
					str := <-lisn.client.write
					_, err := writer.WriteString(str + "\r\n")
					if err != nil {
						fmt.Printf("writec error: %v\n", err)
						break
					}
					fmt.Printf("writec: %s\n", str)
					writer.Flush()
				}
			}()
			go lisn.client.handler()
			break
		}
	}()
	return nil
}

/*func (lisn *IRCListener) Close() {
  lisn.listening = false
  if lisn.client != nil {
    lisn.client.Close()
    lisn.client = nil
  }
  if lisn.listener != nil {
    lisn.listener.Close()
    lisn.listener = nil
  }
}*/
