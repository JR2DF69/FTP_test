package FTPServer

import (
	"FTPServ/FTPAuth"
	"FTPServ/FTPClientConnection"
	"FTPServ/FTPServConfig"
	"FTPServ/Logger"
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

var Config *FTPServConfig.ConfigStorage

var TCPServParameters *TCPServer

type TCPServer struct {
	ServerAddress net.TCPAddr
	Listener      *net.TCPListener
	PeersCount    int
}

func StartFTPServer(cnfg *FTPServConfig.ConfigStorage, users *FTPAuth.Users) {
	Config = cnfg
	TCPServParameters = new(TCPServer)
	err := createTCPSocket()
	if err != nil {
		Logger.Log("func main(): ", err, ". Server stops now")
		os.Exit(1)
	}
	//для сообщения серверу, что соединение закрыто
	FTPConnClosedString := make(chan string)
	//это диспетчер подключений к серверу. Он отслеживает закрытие соединений и контролирует число подключений
	go func() {
		for {
			select {
			case ConnAddr := <-FTPConnClosedString:
				Logger.Log("Closed connection to ", ConnAddr)
				TCPServParameters.PeersCount--
			case <-time.After(2 * time.Second):
				continue
			}
		}
	}()
	//бесконечно пытаемся поймать входящее соединение
	for {
		//а вот и оно
		conn, err := TCPServParameters.Listener.AcceptTCP()
		if err != nil {
			Logger.Log("Connection Listener error: ", err, ". Ignoring connection...")
			continue
		}
		if (TCPServParameters.PeersCount + 1) > Config.MaxClientValue {
			Logger.Log("Max peers value reached. Rejecting connection from ", conn.RemoteAddr())
			conn.Close()
			continue
		}
		FTPConn, err := FTPClientConnection.InitConnection(conn, TCPServParameters.ServerAddress.IP.String(), FTPConnClosedString, Config, users)
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
	TCPServParameters.Listener, err = net.ListenTCP("tcp", &TCPServParameters.ServerAddress)
	if err != nil {
		Logger.Log(fmt.Sprint("GetMachineIPAddress returns error: ", err))
		return errors.New("There was an error while opening TCP Socket")
	}
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
