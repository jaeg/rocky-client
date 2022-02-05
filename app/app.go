package app

import (
	"bufio"
	"context"
	"flag"
	"net"
	"os"
	"strings"
	"time"

	"github.com/jaeg/rocky-client/proxy"
	log "github.com/sirupsen/logrus"
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
	log.SetLevel(log.DebugLevel)
	log.WithField("Name", AppName).Info("Starting")

	a.ConnectToServer()
}

func (a *App) ConnectToServer() {
	for {
		var err error
		a.serverConn, err = net.Dial("tcp", *serverAddress)
		if err != nil {
			log.WithError(err).Error("Error dialing server will retry in 5s")
			time.Sleep(time.Second * 5)
		} else {
			return
		}
	}
}

func (a *App) Run(ctx context.Context) {
	//Run the http server
	go func() {
		serverReader := bufio.NewReader(a.serverConn)
		log.WithField("ServerAddress", a.serverConn.LocalAddr().String()).Infof("Connected to rocky server")

		for {
			select {
			case <-ctx.Done():
				log.Info("Killing thread")
			default:
				message, err := serverReader.ReadString('\n')

				if err != nil {
					log.WithError(err).Error("Error reading message from server")
					if err.Error() == "EOF" {
						os.Exit(1)
					}
					continue
				}

				//Handle message from the proxy server.
				message = strings.Replace(message, "\n", "", -1)
				log.WithField("Message", message).Debug("Message from proxy server")
				if message == "New" {
					id, err := serverReader.ReadString('\n')
					id = strings.Replace(id, "\n", "", -1)
					if err == nil {
						newConnection(a.serverConn, id)
					} else {
						log.WithField("Id", id).WithError(err).Error("Error reading connection information from server")
					}
				}

			}
		}
	}()

	// Handle shutdowns gracefully
	<-ctx.Done()

	log.Info("Client shutdown")
}

//Create a new tunnel to forward traffic from rocky-server to rocky-client's target.
func newConnection(serverConn net.Conn, id string) {
	log.WithField("Id", id).Info("New connection")
	log.WithField("Id", id).Debug("Dial target")
	//Connect to our proxy target
	targetConn, err := net.Dial("tcp", *targetAddress)
	if err != nil {
		log.WithField("Id", id).WithError(err).Error("Error dialing the proxy target")
		return
	}

	log.WithField("Id", id).Debug("Dial server communication port to forward traffic")
	//Open a connection from this client to the rocky server's communication port to start forwarding traffic across it.
	conn, err := net.Dial("tcp", *comunnicationAddress)
	if err != nil {
		log.WithField("Id", id).WithError(err).Error("Error opening proxy communication socket with rocky server")

		return
	}

	log.WithField("Id", id).Debug("Sending connection information with server to identify ourselves")
	//Send some connection information to the server to identify who we are.
	serverConn.Write([]byte(id))
	conn.Write([]byte(id))

	log.WithField("Id", id).Debug("Start thread")
	//Start the proxying
	proxy.NewProxyThread(conn, targetConn)
}
