// FTP Server
package main

import (
	"FTPServ/Config"
	"FTPServ/FTPAuth"
	"FTPServ/Logger"
	"FTPServ/ftpfs"
	"bufio"
	"bytes"
	"io"
	"math"
	"runtime"
	"strconv"
	//"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
)

var config Config.ConfigStorage

var TCPSockParameters TCPSockParams
var Users *FTPAuth.Users

type TCPSockParams struct {
	ServerAddress net.TCPAddr
	Listener      *net.TCPListener
}

type DataConnectionMode uint

const (
	DataConnectionModeActive  DataConnectionMode = iota
	DataConnectionModePassive                    = iota
)

type FTPConnection struct {
	TCPConn              *net.TCPConn
	Writer               *bufio.Writer
	Reader               *bufio.Reader
	DataConnection       FTPDataConnection
	User                 *FTPAuth.User
	TransferType         string
	DataConnectionOpened bool
	FileSystem           ftpfs.FileSystem
}
type FTPDataConnection struct {
	DataPortAddress     net.TCPAddr
	DataConnectionMode  DataConnectionMode
	Connection          *net.TCPConn
	Listener            net.Listener
	Writer              *bufio.Writer
	Reader              *bufio.Reader
	PassiveModeDataConn net.Conn
}

func main() {
	Logger.Log("Starting server>")
	config = Config.LoadConfig()
	err := createTCPSocket()
	if err != nil {
		Logger.Log("func main(): ", err, ". Server stops now")
		return
	}
	Users, err = FTPAuth.LoadUsersList()
	if err != nil {
		Logger.Log("func main(): failed to load users configuration. Server stops now(", err, ")")
		return
	}
	//бесконечно пытаемся поймать входящее соединение
	for {
		//а вот и оно
		conn, _ := TCPSockParameters.Listener.AcceptTCP()
		if err != nil {
			Logger.Log("Connection Listener error: ", err, ". Ignoring connection...")
			continue
		}
		Logger.Log("Got incoming connection from: ", conn.RemoteAddr(), ". Sending 220")

		FTPConn := new(FTPConnection)
		if FTPConn.InitConnection(conn) != nil {
			Logger.Log("Init new connection error: ", err)
			FTPConn = nil
			continue
		}
		go FTPConn.ParseIncomingConnection()
	}
}
func (FTPConn *FTPConnection) InitConnection(Connection *net.TCPConn) error {
	if Connection == nil {
		return errors.New("Connection is nil")
	}
	FTPConn.TCPConn = Connection
	FTPConn.Writer = bufio.NewWriter(Connection)
	FTPConn.Reader = bufio.NewReader(Connection)
	return nil
}
func (FTPConn *FTPConnection) writeMessageToWriter(str string) {
	FTPConn.Writer.WriteString(fmt.Sprint(str, "\r\n"))
	FTPConn.Writer.Flush()
}

func createTCPSocket() error {
	ipaddr, err := GetMachineIPAddress()
	if err != nil {
		Logger.Log(fmt.Sprint("GetMachineIPAddress returns error: ", err))
		return errors.New("There was an error while opening TCP Socket")
	}
	TCPSockParameters.ServerAddress = net.TCPAddr{ipaddr, config.Port, ""}
	Logger.Log(fmt.Sprint("Opening TCP socket at: ", TCPSockParameters.ServerAddress))
	TCPSockParameters.Listener, err = net.ListenTCP("tcp", &TCPSockParameters.ServerAddress)
	if err != nil {
		Logger.Log(fmt.Sprint("GetMachineIPAddress returns error: ", err))
		return errors.New("There was an error while opening TCP Socket")
	}
	Logger.Log(fmt.Sprint("FTP Server running at: ", TCPSockParameters.ServerAddress, "\nWaiting for incoming connections..."))
	return nil
}

func GetMachineIPAddress() (net.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, errors.New(fmt.Sprint("Couldn't get net.InterfaceAddrs: ", err))
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				Logger.Log(fmt.Sprint("GetMachineIPAddress: current machine IP is ", ipnet.IP.To4()))
				return ipnet.IP.To4(), nil
			}
		}
	}
	defer os.Exit(1)
	return nil, errors.New("machine has no IP address. Exiting...")
}
func (FTPConn *FTPConnection) sendResponseToClient(command string, comment interface{}) error {
	defer Logger.Log("Command ", command, " sent to Client")
	switch command {
	case "200":
		FTPConn.writeMessageToWriter(fmt.Sprint("200 ", comment))
	case "215":
		FTPConn.writeMessageToWriter(fmt.Sprint("215 ", "LINUX"))
	case "220":
		FTPConn.writeMessageToWriter("220 Welcome to my Go FTP")
	case "230":
		FTPConn.writeMessageToWriter("230 Logged In")
	case "250":
		FTPConn.writeMessageToWriter(fmt.Sprint("250 ", comment, ""))
	case "257":
		FTPConn.writeMessageToWriter(fmt.Sprint("257 \"", "/", "\" is current root"))
	case "331":
		FTPConn.writeMessageToWriter("331 Password")
	case "530":
		FTPConn.writeMessageToWriter("530 Anonymous denied on server")
	default:
		FTPConn.writeMessageToWriter(fmt.Sprint(command, " ", comment))
	}
	return nil
}
func GetDataPortAddress() string {
	ipAddress := TCPSockParameters.ServerAddress.IP.String()
	ipAddressSplitted := strings.Split(ipAddress, ".")
	dataPortString := CountDataPort()
	ipAddressSplitted = append(ipAddressSplitted, dataPortString[0], dataPortString[1])
	ipAddressPort := strings.Join(ipAddressSplitted, ",")
	return ipAddressPort
}
func CountDataPort() []string {
	part2 := math.Mod(float64(config.DataPort), 256)
	part1 := (float64(config.DataPort) - part2) / 256
	dataPortString := []string{strconv.Itoa(int(part1)), strconv.Itoa(int(part2))}
	return dataPortString
}

func (FTPConn *FTPConnection) CloseConnection() error {
	//close DataConnection
	FTPConn.DataConnection.CloseConnection()
	//check Connection closed
	if FTPConn.TCPConn != nil {
		FTPConn.TCPConn.Close()
		FTPConn.TCPConn = nil
	}
	return nil
}
func (DataConn *FTPDataConnection) Init(ConnectionMode DataConnectionMode, DataPort string) error {
	DataConn.DataConnectionMode = ConnectionMode
	if ConnectionMode == DataConnectionModeActive {
		DataConn.parseDataPortAddr(DataPort)
		Logger.Log(fmt.Sprint("(DataConn *FTPDataConnection) Init(ACTIVE) ACTV ADDRESS: ", DataConn.DataPortAddress))
		return nil
	} else if ConnectionMode == DataConnectionModePassive {
		DataConn.parseDataPortAddr(DataPort)
		Logger.Log(fmt.Sprint("(DataConn *FTPDataConnection) Init(PASSIVE) PASV ADDRESS: ", DataConn.DataPortAddress))
		DataConn.OpenConnection()
		return nil
	}
	return nil
}

func (DataConn *FTPDataConnection) OpenConnection() error {
	if DataConn.CheckConnectionOpened() == true {
		return nil
	}
	if DataConn.DataConnectionMode == DataConnectionModeActive {
		conn, err := net.DialTCP("tcp", nil, &DataConn.DataPortAddress)
		if err != nil {
			Logger.Log("(DataConn *FTPDataConnection)OpenConnection(MODE: active) DialTCP error: ", err)
			DataConn.Connection, DataConn.Writer, DataConn.Reader, DataConn.Listener = nil, nil, nil, nil
			return err
		}
		DataConn.Connection = conn
		DataConn.Listener = nil
		DataConn.Writer = bufio.NewWriter(DataConn.Connection)
		DataConn.Reader = bufio.NewReader(DataConn.Reader)
		/*writer := bufio.NewWriter(listDialer)
		for _, line := range listing {
			writeMessageToWriter(fmt.Sprint(line, "\r\n"), writer)
		}
		listDialer.Close()
		*/
	} else if DataConn.DataConnectionMode == DataConnectionModePassive {
		//OpenDataPort(&connParams)
		Listener, err := net.ListenTCP("tcp", &DataConn.DataPortAddress)
		if err != nil {
			Logger.Log("(DataConn *FTPDataConnection)OpenConnection(MODE: passive) ListenTCP error: ", err)
			DataConn.Connection, DataConn.Writer, DataConn.Reader, DataConn.Listener = nil, nil, nil, nil
			return err
		}
		DataConn.Listener = Listener
	}
	return nil
}
func (DataConn *FTPDataConnection) CheckConnectionOpened() bool {
	if DataConn.DataConnectionMode == DataConnectionModeActive {
		if DataConn.Connection != nil {
			return true
		}
	} else if DataConn.DataConnectionMode == DataConnectionModePassive {
		if DataConn.Listener != nil {
			return true
		}
	}
	return false
}
func (DataConn *FTPDataConnection) CloseConnection() error {
	if DataConn.PassiveModeDataConn != nil {
		DataConn.PassiveModeDataConn.Close()
	}
	if DataConn.Listener != nil {
		DataConn.Listener.Close()
	}
	if DataConn.Connection != nil {
		DataConn.Connection.Close()
	}
	DataConn.PassiveModeDataConn, DataConn.Connection, DataConn.Writer, DataConn.Reader, DataConn.Listener = nil, nil, nil, nil, nil
	Logger.Log("(DataConn *FTPDataConnection) CloseConnection(): all connection were closed")
	return nil
}
func (FTPConn *FTPConnection) ParseIncomingConnection() {
	FTPConn.sendResponseToClient("220", "")
	for {
		reader := make([]byte, 512)
		_, err := FTPConn.Reader.Read(reader)
		if err != nil {
			Logger.Log("parseIncomingConnection, Conn.Read error: ", err, "\r\nConnection closed.")
			FTPConn.CloseConnection()
			return
		}
		reader = bytes.Trim(reader, "\x00")
		input := string(reader)
		commands := strings.Split(input, "\r\n")
		for _, command := range commands {
			Logger.Log(fmt.Sprint("Got command: ", command))
			if strings.TrimSpace(command) == "" {
				continue
			}
			triSymbolCommand := command[:3]
			switch string(triSymbolCommand) {
			case "CCC":
				break
			case "CWD":
				directory := command[5:]
				err := FTPConn.FileSystem.CWD(directory)
				if err != nil {
					if err.Error() == "Not a dir" {
						FTPConn.sendResponseToClient("550", "Not a directory")
						break
					}
					Logger.Log("CWD: ", err)
					FTPConn.sendResponseToClient("550", "Couldn't get directory")
				}
				FTPConn.sendResponseToClient("250", "DirectoryChanged")
				break
			case "ENC":
				break
			case "MFF":
				break
			case "MIC":
				break
			case "MKD":
				break
			case "PWD":
				FTPConn.FileSystem.InitFileSystem(&config)
				FTPConn.sendResponseToClient("257", "/")
				break
			case "RMD":
				break
			}
			if len(command) <= 3 {
				continue
			}
			fourSymbolCommand := command[:4]
			switch string(fourSymbolCommand) {
			case "FEAT":
				FTPConn.sendResponseToClient("211", "-Server feature:\r\n SIZE\r\n211 End")
			case "LIST":
				listing, err := FTPConn.FileSystem.LIST("")
				if err != nil {
					if err.Error() == "Not a dir" {
						FTPConn.sendResponseToClient("550", "Not a directory")
						break
					}
				}
				if FTPConn.DataConnection.OpenConnection() != nil {
					
				}
				FTPConn.sendResponseToClient("150", "Here comes the directory listing")
				if FTPConn.DataConnection.DataConnectionMode == DataConnectionModeActive {
					{
						for _, list := range listing {
							FTPConn.DataConnection.Writer.Write([]byte(fmt.Sprint(list, "\r\n")))
							FTPConn.DataConnection.Writer.Flush()
						}
						FTPConn.sendResponseToClient("226", "Directory sent OK")
						FTPConn.DataConnection.CloseConnection()
						break
					}
				} else if FTPConn.DataConnection.DataConnectionMode == DataConnectionModePassive {
					conn, err := FTPConn.DataConnection.Listener.Accept()
					if err != nil {
						FTPConn.sendResponseToClient("550", "Could not send data")
						break
					}
					writer := bufio.NewWriter(conn)
					for _, line := range listing {
						writer.Write([]byte(fmt.Sprint(line, "\r\n")))
						writer.Flush()
					}
					FTPConn.sendResponseToClient("226", "Directory sent OK")
					conn.Close()
					conn = nil
					FTPConn.DataConnection.CloseConnection()
					break
				}
				//send error message
			case "PASV":
				passPortAddress := GetDataPortAddress()
				if FTPConn.DataConnection.Init(DataConnectionModePassive, passPortAddress) != nil {
					break
				}
				FTPConn.sendResponseToClient("227", fmt.Sprint("Entering Passive Mode (", passPortAddress, ").\r\n"))
			case "TYPE":
				sendType := command[5:]
				FTPConn.TransferType = sendType
				FTPConn.sendResponseToClient("200", "Set type successful!")
			case "SIZE":
				path := command[5:]
				size, err := FTPConn.FileSystem.GetFileSize(path)
				if err != nil {
					FTPConn.sendResponseToClient("550", "Could not get file size")
					break
				}
				FTPConn.sendResponseToClient("213", size)
			case "USER":
				//new user
				userName := bytes.Trim(reader[5:], "\n")
				userName = bytes.Trim(userName, "\r")
				userNameStr := string(userName)
				if strings.ToLower(userNameStr) == "anonymous" {
					Logger.Log("This is anonymous!")
					if config.Anonymous == false {
						FTPConn.sendResponseToClient("530", "")
						FTPConn.CloseConnection()
						break
					}
				}
				user := Users.CheckUserName(userNameStr)
				if user == nil {
					Logger.Log("Command \"USER\": wrong user name!")
					FTPConn.sendResponseToClient("430", "Wrong username or password")
					break
				}
				FTPConn.User = user
				FTPConn.sendResponseToClient("331", "")
				break
			case "PASS":
				pswd := command[5:]
				if FTPConn.User == nil {
					FTPConn.sendResponseToClient("430", "Wrong username or password")
					break
				}
				if FTPConn.User.CheckPswd(pswd) == false {
					FTPConn.sendResponseToClient("430", "Wrong username or password")
					break
				}
				FTPConn.sendResponseToClient("230", "")
				//new pass
				break
			case "PORT":
				Logger.Log("PORT sent to Server")
				Port := command[5:]
				FTPConn.DataConnection.Init(DataConnectionModeActive, Port)
				FTPConn.sendResponseToClient("200", fmt.Sprint("PORT command done", FTPConn.DataConnection.DataPortAddress))
			case "RETR":
				fileName := command[5:]
				file, err := FTPConn.FileSystem.RETR(fileName)
				if err != nil {
					Logger.Log("RETR Command, fsRETR error: ", err)
					FTPConn.sendResponseToClient("550", "File transfer error")
					break
				}
				FTPConn.sendResponseToClient("150", fmt.Sprint("Opening binary stream for", fileName))
				sendFileBuff := make([]byte, config.BufferSize)
				for {
					_, err := file.Read(sendFileBuff)
					if err == io.EOF {
						break
					}
					err = FTPConn.DataConnection.sendBinaryData(sendFileBuff)
					if err != nil {
						Logger.Log("RETR Command, File transfer error: ", err)
						FTPConn.sendResponseToClient("550", "File transfer error")
						break
					}
				}
				FTPConn.DataConnection.CloseConnection()
				FTPConn.sendResponseToClient("226", "Transfer complete")
			case "SYST":
				FTPConn.sendResponseToClient("215", runtime.GOOS)
			case "QUIT":
				Logger.Log("closed connection")
				FTPConn.CloseConnection()
				break
			}
		}
	}
}
func (DataConn *FTPDataConnection) parseDataPortAddr(dataPort string) {
	PortParamsSplitted := strings.Split(dataPort, ",")
	num1, _ := strconv.Atoi(PortParamsSplitted[4])
	num2, _ := strconv.Atoi(PortParamsSplitted[5])
	portnum := num1*256 + num2
	ip := net.ParseIP(fmt.Sprint(PortParamsSplitted[0], ".", PortParamsSplitted[1], ".", PortParamsSplitted[2], ".", PortParamsSplitted[3]))
	DataConn.DataPortAddress = net.TCPAddr{ip, portnum, ""}
}
func (DataConn *FTPDataConnection) sendBinaryData(dataBytes []byte) error {
	/*if DataConn.CheckConnectionOpened() == false {
		DataConn.OpenConnection()
	}*/
	if DataConn.DataConnectionMode == DataConnectionModeActive {
		DataConn.Connection.Write(dataBytes)
	} else if DataConn.DataConnectionMode == DataConnectionModePassive {
		if DataConn.PassiveModeDataConn != nil {
			DataConn.PassiveModeDataConn.Write(dataBytes)
			return nil
		}
		conn, err := DataConn.Listener.Accept()
		if err != nil {
			return err
		}
		conn.Write(dataBytes)
		DataConn.PassiveModeDataConn = conn
	}
	return nil
}
