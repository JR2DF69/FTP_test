package FTPServer

import (
	"FTPServ/FTPAuth"
	"FTPServ/FTPClientConnection"
	"FTPServ/FTPServConfig"
	"FTPServ/FTPtls"
	"FTPServ/Logger"
	//"FTPServ/CBModule"
	"errors"
	"fmt"
	"net"
	"os"
)

var Config *FTPServConfig.ConfigStorage

var TCPServParameters *TCPServer

type TCPServer struct {
	ServerAddress net.TCPAddr
	Listener      net.Listener
	PeersCount    int
	TLSConfig     *FTPtls.FTPTLSServerParameters
}

func StartFTPServer(cnfg *FTPServConfig.ConfigStorage, users *FTPAuth.Users, stopCh chan bool) {
	Config = cnfg
	TCPServParameters = new(TCPServer)
	//для сообщения серверу, что соединение закрыто
	FTPConnClosedString := make(chan string)
	//generate config for server
	params, err := FTPtls.ReadNewTLSConfig()
	if err != nil {
		Logger.Log("Parse tls config error: ", err)
	} else {
		Logger.Log("TLS config loaded successfully. Clients can use AUTH command for TLS connection!")
		TCPServParameters.TLSConfig = params
	}
	err = createTCPSocket()
	if err != nil {
		Logger.Log("func main(): ", err, ". Server stops now")
		os.Exit(1)
	}
	/*if err = CBModule.InitNewConnection(); err != nil{
		Logger.Log("Error while initializing CB bridge: ", err)
	}*/
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
		//а вот и оно
		/*select {
		case StopServer := <-stopCh:
			if StopServer {
				return
			}
		}*/
		conn, err := TCPServParameters.Listener.Accept()
		if err != nil {
			Logger.Log("Connection Listener error: ", err, ". Ignoring connection...")
			continue
		}
		if (TCPServParameters.PeersCount + 1) > Config.MaxClientValue {
			Logger.Log("Max peers value reached. Rejecting connection from ", conn.RemoteAddr())
			conn.Close()
			continue
		}
		/*err = TLSConn.Handshake()
		if err != nil {
			Logger.Log("Handshake error: ", err)
		}*/
		//TLSConn.Write([]byte("220 Welcome"))
		FTPConn, err := FTPClientConnection.InitConnection(conn, TCPServParameters.ServerAddress.IP.String(), FTPConnClosedString, Config, users, TCPServParameters.TLSConfig)
		if err != nil {
			Logger.Log("Init new connection error: ", err)
			FTPConn = nil
			continue
		}
		TCPServParameters.PeersCount++
		Logger.Log("Got incoming connection from: ", conn.RemoteAddr(), ". Sending 220")
		go FTPConn.ParseIncomingConnection()
	}
}
func createTCPSocket() error {
	ipaddr, err := getMachineIPAddress()
	if err != nil {
		Logger.Log(fmt.Sprint("GetMachineIPAddress returns error: ", err))
		return errors.New("There was an error while opening TCP Socket")
	}
	TCPServParameters.ServerAddress = net.TCPAddr{ipaddr, Config.Port, ""}
	Logger.Log(fmt.Sprint("Opening TCP socket at: ", TCPServParameters.ServerAddress))
	Listener, err := net.Listen("tcp4", TCPServParameters.ServerAddress.String())
	if err != nil {
		Logger.Log("Error to listen to TCP: ", err)
		return errors.New("There was an error while opening TCP Socket")
	}
	TCPServParameters.Listener = Listener
	//TCPServParameters.Listener = tls.NewListener(Listener, TCPServParameters.TLSConfig)
	Logger.Log(fmt.Sprint("FTP Server running at: ", TCPServParameters.ServerAddress, "\nWaiting for incoming connections..."))
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
