package FTPClientConnection

import (
	"FTPServ/FTPAuth"
	"FTPServ/FTPDataTransfer"
	"FTPServ/FTPServConfig"
	"FTPServ/Logger"
	"FTPServ/ftpfs"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"runtime"
	"strings"
)

type FTPConnection struct {
	TCPConn              *net.TCPConn
	FTPConnClosedString  chan string //channel to say server conn closed
	Writer               *bufio.Writer
	Reader               *bufio.Reader
	DataConnection       *FTPDataTransfer.FTPDataConnection
	User                 *FTPAuth.User
	TransferType         string
	DataConnectionOpened bool
	FileSystem           ftpfs.FileSystem
	GlobalConfig         *FTPServConfig.ConfigStorage
	ServerAddress        string
}

var users *FTPAuth.Users

func InitConnection(Connection *net.TCPConn, serverAddr string, EndConnChannel chan string, ServerConfig *FTPServConfig.ConfigStorage, Users *FTPAuth.Users) (*FTPConnection, error) {
	FTPConn := new(FTPConnection)
	if Connection == nil {
		return nil, errors.New("Connection is nil")
	}
	FTPConn.TCPConn = Connection
	FTPConn.Writer = bufio.NewWriter(Connection)
	FTPConn.Reader = bufio.NewReader(Connection)
	FTPConn.FTPConnClosedString = EndConnChannel
	FTPConn.GlobalConfig = ServerConfig
	users = Users
	FTPConn.ServerAddress = serverAddr
	dc, err := FTPDataTransfer.NewConnection(serverAddr, ServerConfig)
	if err != nil {
		return nil, err
	}
	FTPConn.DataConnection = dc
	return FTPConn, nil
}
func (FTPConn *FTPConnection) writeMessageToWriter(str string) {
	FTPConn.Writer.WriteString(fmt.Sprint(str, "\r\n"))
	FTPConn.Writer.Flush()
}
func (FTPConn *FTPConnection) sendResponseToClient(command string, comment interface{}) error {
	defer Logger.Log("Command ", command, " sent to Client")
	switch command {
	case "200":
		FTPConn.writeMessageToWriter(fmt.Sprint("200 ", comment))
	case "211":
		FTPConn.writeMessageToWriter(fmt.Sprint("211", comment))
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

func (FTPConn *FTPConnection) CloseConnection(TCPClosed bool) error {
	//close DataConnection
	//FTPConn.DataConnection.CloseConnection()
	//check Connection closed
	_ = 1
	fmt.Print("Closing connection...")
	if FTPConn.DataConnection != nil {
		FTPConn.DataConnection.CloseConnection()
		FTPConn.DataConnection = nil
	}
	FTPConn.FTPConnClosedString <- FTPConn.TCPConn.RemoteAddr().String()
	if FTPConn.TCPConn != nil {
		err := FTPConn.TCPConn.Close()
		if err == nil {
			FTPConn.TCPConn = nil
		}
	}
	Logger.Log("Connection closed")
	return nil
}

func (FTPConn *FTPConnection) ParseIncomingConnection() {
	FTPConn.sendResponseToClient("220", "")
	for {
		reader := make([]byte, 512)
		_, err := FTPConn.Reader.Read(reader)
		if err != nil {
			Logger.Log("parseIncomingConnection, Conn.Read error: ", err, "\r\nConnection closed.")
			//FTPConn.CloseConnection(false)
			return
		}
		reader = bytes.Trim(reader, "\x00")
		input := string(reader)
		commands := strings.Split(input, "\r\n")
		for _, command := range commands {
			if len(strings.TrimSpace(command)) == 0 {
				continue
			}
			Logger.Log(fmt.Sprint("Got command: ", command))
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
				FTPConn.FileSystem.InitFileSystem(FTPConn.GlobalConfig)
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
				FTPConn.sendResponseToClient("211", "-Server feature:\r\n SIZE\r\n211 END")
			case "LIST":
				listing, err := FTPConn.FileSystem.LIST("")
				if err != nil {
					if err.Error() == "Not a dir" {
						FTPConn.sendResponseToClient("550", "Not a directory")
						break
					}
				}
				FTPConn.sendResponseToClient("150", "Here comes the directory listing")
				sendingdir := strings.Join(listing, "\r\n")
				err = FTPConn.DataConnection.TransferASCIIData(sendingdir)
				if err != nil {
					FTPConn.sendResponseToClient("550", "Could not send data")
					Logger.Log("Couln't send LIST data: ", err)
					break
				}
				FTPConn.sendResponseToClient("226", "Directory sent OK")
				break
				//send error message*/
			case "PASV":
				passPortAddress, err := FTPConn.DataConnection.InitPassiveConnection()
				if err != nil {
					Logger.Log("PASV: couldn't open passive port...", err)
					FTPConn.sendResponseToClient("425", "PASV start error...")
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
				Logger.Log(size)
				FTPConn.sendResponseToClient("213", size)
			case "USER":
				//new user
				userName := bytes.Trim(reader[5:], "\n")
				userName = bytes.Trim(userName, "\r")
				userNameStr := string(userName)
				if strings.ToLower(userNameStr) == "anonymous" {
					Logger.Log("This is anonymous!")
					if FTPConn.GlobalConfig.Anonymous == false {
						FTPConn.sendResponseToClient("530", "")
						//FTPConn.CloseConnection()
						break
					} else {
						FTPConn.sendResponseToClient("230", "")
						break
					}
				}
				user := users.CheckUserName(userNameStr)
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
				port := command[5:]
				err := FTPConn.DataConnection.InitActiveConnection(port)
				if err != nil {
					FTPConn.sendResponseToClient("550", fmt.Sprint("Dialing active port error: ", err))
					break
				}
				FTPConn.sendResponseToClient("200", fmt.Sprint("PORT command done", FTPConn.DataConnection.FTPActiveDataConnection.DataPortAddress.String()))
			case "RETR":
				fileName := command[5:]
				file, err := FTPConn.FileSystem.RETR(fileName)
				if err != nil {
					Logger.Log("RETR Command, fsRETR error: ", err)
					FTPConn.sendResponseToClient("550", "File transfer error")
					break
				}
				FTPConn.sendResponseToClient("150", fmt.Sprint("Opening binary stream for", fileName))
				err = FTPConn.DataConnection.TransferBinaryFile(file)
				if err != nil {
					Logger.Log("RETR command error: ", err)
					FTPConn.sendResponseToClient("550", "File transfer error")
					break
				}
				FTPConn.sendResponseToClient("226", "Transfer complete")
			case "SYST":
				FTPConn.sendResponseToClient("215", runtime.GOOS)
			case "QUIT":
				Logger.Log("Closing connection")
				FTPConn.CloseConnection(true)
				return
			}
		}
	}
}
