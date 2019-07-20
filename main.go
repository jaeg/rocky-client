package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"strings"
)

var target = flag.String("target", "localhost:6379", "Target address to forward traffic to.")
var server = flag.String("server", "localhost:9999", "Server location")
var proxyPort = flag.String("proxy", "localhost:9998", "Port the proxy connection takes place on.")

func main() {
	flag.Parse()

	serverConn, err := net.Dial("tcp", *server)
	if err != nil {
		fmt.Println(err)
		return
	}
	serverReader := bufio.NewReader(serverConn)

	for {
		message, err := serverReader.ReadString('\n')
		if err == nil {
			message = strings.Replace(message, "\n", "", -1)
			fmt.Println("Message", message)
			if message == "New" {
				id, err := serverReader.ReadString('\n')
				id = strings.Replace(id, "\n", "", -1)
				fmt.Println("ID", id)
				if err == nil {
					newConnection(serverConn, id)
				}
			}
		}
	}

}

func newConnection(serverConn net.Conn, id string) {
	fmt.Println("New connection")
	targetConn, err := net.Dial("tcp", *target)
	if err != nil {
		fmt.Println(err)
		return
	}

	conn, err := net.Dial("tcp", *proxyPort)
	if err != nil {
		fmt.Println(err)
		return
	}
	serverConn.Write([]byte(id))
	conn.Write([]byte(id))
	go handleToTarget(conn, targetConn)
	go handleFromTarget(conn, targetConn)

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
		if err != nil {
			fmt.Println("Error reading from target:", err.Error())
			return
		}
		conn.Write(buf)
	}
}
