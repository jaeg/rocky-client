package app

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	"github.com/jaeg/rocky-client/proxy"
	log "github.com/sirupsen/logrus"
)

const AppName = "rocky-client"

var communicationCertFilePath = flag.String("communication-cert", "certs/client.pem", "location of cert file")
var communicationKeyFilePath = flag.String("communication-key", "certs/client.key", "location of key file")
var communicationCAFilePath = flag.String("communication-ca", "certs/ca.crt", "location of ca file")

var targetAddress = flag.String("target", "localhost:8090", "Target address to forward traffic to.")
var serverAddress = flag.String("server", "localhost:9999", "Server location")
var tunnelAddress = flag.String("proxy", "localhost:9998", "Port the proxy connection takes place on.")

type App struct {
	connections            map[string]net.Conn
	communicationTLSConfig *tls.Config
}

func (a *App) Init() error {
	a.connections = make(map[string]net.Conn)
	flag.Parse()

	//Start the logger
	log.SetLevel(log.DebugLevel)
	log.WithField("Name", AppName).Info("Starting")

	if *communicationCAFilePath != "" {
		err := a.LoadCommunicationCerts()
		if err != nil {
			log.WithError(err).Fatal("Failed loading certs")
			return err
		}
	}

	return nil
}

func (a *App) ConnectToServer() (net.Conn, error) {

	var err error

	var serverConn net.Conn
	if a.communicationTLSConfig != nil {
		serverConn, err = tls.Dial("tcp", *serverAddress, a.communicationTLSConfig)
		if err != nil {
			log.WithError(err).Error("Error opening proxy communication socket with rocky server")
		}
	} else {
		serverConn, err = net.Dial("tcp", *serverAddress)
		if err != nil {
			log.WithError(err).Error("Error opening server communication socket with rocky server")
		}
	}

	if err != nil {
		return nil, err

	}
	log.WithField("ServerAddress", serverConn.LocalAddr().String()).Infof("Connected to rocky server")
	return serverConn, nil
}

func (a *App) LoadCommunicationCerts() error {
	caCert, err := ioutil.ReadFile(*communicationCAFilePath)
	if err != nil {
		log.Error(err)
		return err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cer, err := tls.LoadX509KeyPair(*communicationCertFilePath, *communicationKeyFilePath)

	if err != nil {
		log.Error(err)
		return err
	}

	a.communicationTLSConfig = &tls.Config{Certificates: []tls.Certificate{cer}, RootCAs: caCertPool, InsecureSkipVerify: false}

	return nil
}

func (a *App) Run(ctx context.Context) {
	go func() {
		for {
			//Open connection to rocky proxy to get something to proxy.
			serverConn, err := a.ConnectToServer()
			if err != nil {
				log.WithError(err).Error("Failed to connect to rocky server. Will retry in 1s")
				time.Sleep(time.Second)
				continue
			}

			serverReader := bufio.NewReader(serverConn)
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
						a.newConnection(serverConn, id)
					} else {
						log.WithField("Id", id).WithError(err).Error("Error reading connection information from server")
					}
				}
			}

			serverConn.Close()
		}
	}()

	// Handle shutdowns gracefully
	<-ctx.Done()

	log.Info("Client shutdown")
}

//Create a new tunnel to forward traffic from rocky-server to rocky-client's target.
func (a *App) newConnection(serverConn net.Conn, id string) error {
	log.WithField("Id", id).Info("New connection")
	log.WithField("Id", id).Debug("Dial target")
	//Connect to our proxy target
	targetConn, err := net.Dial("tcp", *targetAddress)
	if err != nil {
		log.WithField("Id", id).WithError(err).Error("Error dialing the proxy target")
		return err
	}

	//Open a connection from this client to the rocky server's communication port to start forwarding traffic across it.
	var conn net.Conn
	if a.communicationTLSConfig != nil {
		log.WithField("Id", id).Debug("Dial server tunnel port to forward traffic with tls")
		conn, err = tls.Dial("tcp", *tunnelAddress, a.communicationTLSConfig)
		if err != nil {
			log.WithField("Id", id).WithError(err).Error("Error opening proxy communication socket with rocky server")

			return err
		}
	} else {
		log.WithField("Id", id).Debug("Dial server tunnel port to forward traffic")
		conn, err = net.Dial("tcp", *tunnelAddress)
		if err != nil {
			log.WithField("Id", id).WithError(err).Error("Error opening proxy communication socket with rocky server")

			return err
		}
	}

	log.WithField("Id", id).Debug("Sending connection information with server to identify ourselves")
	//Send some connection information to the server to identify who we are.
	serverConn.Write([]byte(id))
	conn.Write([]byte(id))

	log.WithField("Id", id).Debug("Start thread")
	//Start the proxying
	proxy.NewProxyThread(id, conn, targetConn)

	return nil
}
