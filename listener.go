package main

import (
	"fmt"
	"net"
	"strconv"
)

type IRCListener struct {
	listening bool
	addr      *net.TCPAddr
}

func CreateListener(hostname string, port int) (*IRCListener, error) {
	addr, err := net.ResolveTCPAddr("tcp4", hostname+":"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}
	return &IRCListener{false, addr}, nil
}

func (lisn *IRCListener) Listen() error {
	if lisn.listening {
		return nil
	}

	listener, err := net.ListenTCP("tcp4", lisn.addr)
	if err != nil {
		return err
	}

	lisn.listening = true

	fmt.Printf("Listening for TCP on %d\n", lisn.addr.Port)

	for lisn.listening {
		conn, err := listener.AcceptTCP()
		if err != nil {
			fmt.Printf("Listen error: %v\n", err)
			continue
		}

		fmt.Printf("Got connection from %s\n", conn.RemoteAddr().String())

		AcceptAuthConnection(conn)
	}
	return nil
}
