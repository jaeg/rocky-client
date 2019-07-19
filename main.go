package main

import (
	"flag"
	"fmt"
	"net"
)

var target = flag.String("target", "localhost:6379", "Target address to forward traffic to.")
var server = flag.String("server", "localhost:9999", "Server location")

func main() {
	flag.Parse()

	serverConn, err := net.Dial("tcp", *server)
	if err != nil {
		fmt.Println(err)
		return
	}

	targetConn, err := net.Dial("tcp", *target)
	if err != nil {
		fmt.Println(err)
		return
	}

	go handleToTarget(serverConn, targetConn)
	go handleFromTarget(serverConn, targetConn)

	for {
	}
}

func handleToTarget(conn net.Conn, targetConn net.Conn) {
	for {
		buf := make([]byte, 1)
		_, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading to target:", err.Error())
			return
		}

		targetConn.Write(buf)
	}
}

func handleFromTarget(conn net.Conn, targetConn net.Conn) {
	for {
		buf := make([]byte, 1)
		_, err := targetConn.Read(buf)
		//stringed := string(buf)
		//fmt.Println("From target", stringed+"<end>")
		if err != nil {
			fmt.Println("Error reading from target:", err.Error())
			return
		}
		conn.Write(buf)
	}
}
