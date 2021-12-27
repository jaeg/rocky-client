package app

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/google/logger"
)

const AppName = "rocker-client"

var certFile = flag.String("cert-file", "", "location of cert file")
var keyFile = flag.String("key-file", "", "location of key file")
var logPath = flag.String("log-path", "./logs.txt", "Logs location")

var targetAddress = flag.String("target", "localhost:8090", "Target address to forward traffic to.")
var serverAddress = flag.String("server", "localhost:9999", "Server location")
var comunnicationAddress = flag.String("proxy", "localhost:9998", "Port the proxy connection takes place on.")

type App struct {
	connections map[string]net.Conn
	serverConn  net.Conn
}

func (a *App) Init() {
	a.connections = make(map[string]net.Conn)
	flag.Parse()

	//Start the logger
	lf, err := os.OpenFile(*logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		logger.Fatalf("Failed to open log file: %v", err)
	}

	logger.Init(AppName, true, true, lf)

	logger.Infof("%s Starting", AppName)

	a.serverConn, err = net.Dial("tcp", *serverAddress)
	if err != nil {
		logger.Errorf("Error dialing server %s", err.Error())
		return
	}
}

func (a *App) Run(ctx context.Context) {
	defer logger.Close()
	//Run the http server
	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("Killing thread")
			default:

				serverReader := bufio.NewReader(a.serverConn)
				logger.Infof("Connected to rocky server %s", a.serverConn.LocalAddr().String())

				for {
					message, err := serverReader.ReadString('\n')

					if err == nil {
						message = strings.Replace(message, "\n", "", -1)
						logger.Infof("Message from rocky server: %s", message)
						if message == "New" {
							id, err := serverReader.ReadString('\n')
							id = strings.Replace(id, "\n", "", -1)
							fmt.Println("ID", id)
							if err == nil {
								newConnection(a.serverConn, id)
							} else {
								logger.Errorf("Error reading connection information from server:  %s", err.Error())
							}
						}
					} else {
						logger.Errorf("Error reading message from server %s", err.Error())
						if err.Error() == "EOF" {
							os.Exit(1)
						}
					}
				}
			}
		}
	}()

	// Handle shutdowns gracefully
	<-ctx.Done()

	logger.Info("Client shutdown")
}

//Create a new connection to forward traffic from rocky-server to rocky-client's target.
func newConnection(serverConn net.Conn, id string) {
	logger.Info("New connection")
	//Connect to our proxy target
	targetConn, err := net.Dial("tcp", *targetAddress)
	if err != nil {
		logger.Errorf("Error dialing the proxy target: %s", err.Error())
		return
	}

	//Open a connection from this client to the rocky server's communication port to start forwarding traffic across it.
	conn, err := net.Dial("tcp", *comunnicationAddress)
	if err != nil {
		logger.Errorf("Error opening proxy communication socket with rocky server %s", err.Error())

		return
	}
	//Send some connection information to the server to identify who we are.
	serverConn.Write([]byte(id))
	conn.Write([]byte(id))

	//Start the proxying
	go handleToTarget(conn, targetConn)
	go handleFromTarget(conn, targetConn)

}

func handleToTarget(conn net.Conn, targetConn net.Conn) {
	for {
		buf := make([]byte, 1)
		_, err := conn.Read(buf)
		if err != nil {
			logger.Errorf("Error reading to send to target: %s", err.Error())
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
			logger.Errorf("Error reading data from the target to forward to rocky server: %s", err.Error())
			return
		}
		conn.Write(buf)
	}
}
