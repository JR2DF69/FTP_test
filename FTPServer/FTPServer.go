package FTPServer

import (
	"FTPServ/CBModule"
	"FTPServ/FTPAuth"
	"FTPServ/FTPClientConnection"
	"FTPServ/FTPServConfig"
	"FTPServ/FTPtls"
	"FTPServ/Logger"
	"crypto/tls"

	//"FTPServ/CBModule"
	"errors"
	"fmt"
	"net"
	"os"
)

var Config *FTPServConfig.ConfigStorage

type TCPServer struct {
	ServerAddress net.TCPAddr
	Listener      net.Listener
	PeersCount    uint
	TLSConfig     *FTPtls.FTPTLSServerParameters
}

func StartFTPServer(cnfg *FTPServConfig.ConfigStorage, users *FTPAuth.Users, stopCh chan bool, Secured bool) {
	Config = cnfg
	TCPServParameters := new(TCPServer)
	//для сообщения серверу, что соединение закрыто
	FTPConnClosedString := make(chan string)
	//generate config for server
	params, err := FTPtls.ReadNewTLSConfig()
	if err != nil {
		Logger.Log("Parse tls config error: ", err)
		if Secured {
			fmt.Print("Server stops now...")
			os.Exit(1)
		}
	} else {
		if Secured {
			Logger.Log("TLS config loaded successfully. No need to use AUTH command (FTPS server is running now)")
		} else {
			Logger.Log("TLS config loaded successfully. Clients can use AUTH command for TLS connection!")
		}
		TCPServParameters.TLSConfig = params
	}
	err = TCPServParameters.CreateTCPSocket(Secured)
	if err != nil {
		Logger.Log("func main(): ", err, ". Server stops now")
		os.Exit(1)
	}
	if err = CBModule.InitNewConnection(); err != nil {
		Logger.Log("Error while initializing CB bridge: ", err)
	}
	//это диспетчер подключений к серверу. Он отслеживает закрытие соединений и контролирует число подключений
	//также через него отслеживаем команды об остановке сервера
	go func() {
		for {
			select {
			case ConnAddr := <-FTPConnClosedString:
				Logger.Log("Closed connection to ", ConnAddr)
				TCPServParameters.PeersCount--
			case StopServer := <-stopCh:
				if StopServer {
					Logger.Log("Stopping server...")
					return
				}
			}
		}
	}()
	defer TCPServParameters.Listener.Close()
	//бесконечно пытаемся поймать входящее соединение
	for {
		conn, err := TCPServParameters.Listener.Accept()
		if err != nil {
			Logger.Log("Connection Listener error: ", err, ". Ignoring connection...")
			continue
		}
		if (int(TCPServParameters.PeersCount) + 1) > int(Config.MaxClientValue) {
			Logger.Log("Max peers value reached. Rejecting connection from ", conn.RemoteAddr())
			conn.Close()
			continue
		}
		FTPConn, err := FTPClientConnection.InitConnection(conn, TCPServParameters.ServerAddress.IP.String(), FTPConnClosedString, Config, users, TCPServParameters.TLSConfig, (TCPServParameters.PeersCount + 1))
		if err != nil {
			Logger.Log("Init new connection error: ", err)
			FTPConn = nil
			continue
		}
		FTPConn.UsingTLS = Secured
		TCPServParameters.PeersCount++
		Logger.Log("Got incoming connection from: ", conn.RemoteAddr(), ". Sending 220")
		go FTPConn.ParseIncomingConnection()
	}
}
func (s *TCPServer) CreateTCPSocket(secured bool) error {
	ipaddr, err := getMachineIPAddress()
	if err != nil {
		Logger.Log(fmt.Sprint("GetMachineIPAddress returns error: ", err))
		return errors.New("There was an error while opening TCP Socket")
	}
	s.ServerAddress = net.TCPAddr{ipaddr, Config.Port, ""}
	Logger.Log(fmt.Sprint("Opening TCP socket at: ", s.ServerAddress), "(secured: ", secured, ")")
	var Listener net.Listener
	if secured {
		Listener, err = tls.Listen("tcp4", s.ServerAddress.String(), s.TLSConfig.TLSConfig)
		if err != nil {
			Logger.Log("Error to listen to TCP (secured): ", err)
			return errors.New("There was an error while opening TCP-TLS Socket")
		}
	} else {
		Listener, err = net.Listen("tcp4", s.ServerAddress.String())
		if err != nil {
			Logger.Log("Error to listen to TCP: ", err)
			return errors.New("There was an error while opening TCP Socket")
		}
	}
	s.Listener = Listener
	//TCPServParameters.Listener = tls.NewListener(Listener, TCPServParameters.TLSConfig)
	Logger.Log(fmt.Sprint("FTP Server running at: ", s.ServerAddress, "(secured : ", secured, ").", "\nWaiting for incoming connections..."))
	return nil
}

func getMachineIPAddress() (net.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, errors.New(fmt.Sprint("Couldn't get net.InterfaceAddrs: ", err))
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				Logger.Log(fmt.Sprint("Current machine IP is ", ipnet.IP.To4()))
				return ipnet.IP.To4(), nil
			}
		}
	}
	defer os.Exit(1)
	return nil, errors.New("machine has no IP address. Exiting...")
}
