package FTPClientConnection

import (
	//"FTPServ/CBModule"
	"FTPServ/FTPAuth"
	"FTPServ/FTPDataTransfer"
	"FTPServ/FTPServConfig"
	"FTPServ/FTPtls"
	"FTPServ/Logger"
	"FTPServ/ftpfs"
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"runtime"
	"strings"
	//"time"
)

type FTPConnection struct {
	TCPConn              net.Conn
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
	actionBuffer         FTPConnectionBuffer
	TLSConfig            *FTPtls.FTPTLSServerParameters
	UsingTLS             bool
}
type FTPConnectionBuffer struct {
	RenameObj *ftpfs.RenameableObj
}

var users *FTPAuth.Users

func InitConnection(Connection net.Conn, serverAddr string, EndConnChannel chan string, ServerConfig *FTPServConfig.ConfigStorage, Users *FTPAuth.Users, TLSConfig *FTPtls.FTPTLSServerParameters) (*FTPConnection, error) {
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
	FTPConn.TLSConfig = TLSConfig
	return FTPConn, nil
}
func (FTPConn *FTPConnection) writeMessageToWriter(str string) {
	FTPConn.Writer.WriteString(fmt.Sprint(str, "\r\n"))
	err := FTPConn.Writer.Flush()
	if err != nil {
		Logger.Log("Error to flush writer: ", err)
	}
	/*_, err := FTPConn.TCPConn.Write([]byte(fmt.Sprint(str, "\r\n")))
	if err != nil {
		Logger.Log(err)
	}*/
}
func (FTPConn *FTPConnection) sendResponseToClient(command string, comment interface{}) error {
	defer Logger.Log("Command ", command, " sent to Client")
	switch command {
	case "200":
		FTPConn.writeMessageToWriter(fmt.Sprint("200 ", comment))
	case "211":
		fallthrough
	case "213":
		FTPConn.writeMessageToWriter(fmt.Sprint(command, comment))
		break
	case "215":
		FTPConn.writeMessageToWriter(fmt.Sprint("215 ", "UNIX TYPE: L8"))
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
		FTPConn.writeMessageToWriter("530 Not logged in")
	default:
		FTPConn.writeMessageToWriter(fmt.Sprint(command, " ", comment))
	}
	return nil
}
func (FTPConn *FTPConnection) IsAuthenticated() bool {
	return FTPConn.User != nil
}
func (FTPConn *FTPConnection) CloseConnection(TCPClosed bool) error {
	//close DataConnection
	//FTPConn.DataConnection.CloseConnection()
	//check Connection closed
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
func (FTPConn *FTPConnection) InitTLSConnection() error {
	conn := tls.Server(FTPConn.TCPConn, FTPConn.TLSConfig.TLSConfig)
	if conn == nil {
		return errors.New("Couldn't serve TLS connection")
	}
	err := conn.Handshake()
	if err != nil {
		return err
	}
	FTPConn.TCPConn = conn
	FTPConn.Reader = bufio.NewReader(conn)
	FTPConn.Writer = bufio.NewWriter(conn)
	return nil
}
func (FTPConn *FTPConnection) ParseIncomingConnection() {
	FTPConn.sendResponseToClient("220", "")
	for {
		reader := make([]byte, 512)
		_, err := FTPConn.TCPConn.Read(reader)
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
				if FTPConn.IsAuthenticated() == false {
					FTPConn.sendResponseToClient("530", "Not logged in")
					break
				}
				if len(command) <= 3 {
					FTPConn.sendResponseToClient("550", "No path specified")
					break
				}
				directory := command[4:]
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
				if len(command) <= 3 {
					FTPConn.sendResponseToClient("550", "No directory name in args")
					break
				}
				dirName := command[4:]
				err := FTPConn.FileSystem.MakeDir(dirName)
				if err != nil {
					FTPConn.sendResponseToClient("550", "Couldn't create specified directory")
					break
				}
				FTPConn.sendResponseToClient("250", fmt.Sprint("Directory ", dirName, " created!"))
				break
			case "PWD":
				if FTPConn.IsAuthenticated() == false {
					FTPConn.sendResponseToClient("530", "Not logged in")
					break
				}
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
				FTPConn.sendResponseToClient("211", "-Server feature:\r\n SIZE\r\n AUTH\r\n STOR\r\n211 END")
			case "LIST":
				if FTPConn.IsAuthenticated() == false {
					FTPConn.sendResponseToClient("530", "Not logged in")
					break
				}
				var key string
				if len(command) >= 7 {
					key = command[6:]
				}
				_ = key
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
					FTPConn.DataConnection.CloseConnection()
					FTPConn.sendResponseToClient("550", "Could not send data")
					Logger.Log("Couldn't send LIST data (key -l): ", err)
					break
				}
				FTPConn.sendResponseToClient("226", "Directory sent OK")
				break
			case "PASV":
				if FTPConn.IsAuthenticated() == false {
					FTPConn.sendResponseToClient("530", "Not logged in")
					break
				}
				FTPConn.DataConnection.UsingTLS = FTPConn.UsingTLS
				FTPConn.DataConnection.TLSConfig = FTPConn.TLSConfig
				passPortAddress, err := FTPConn.DataConnection.InitPassiveConnection()
				if err != nil {
					Logger.Log("PASV: couldn't open passive port...", err)
					FTPConn.sendResponseToClient("425", "PASV start error...")
					break
				}
				FTPConn.sendResponseToClient("227", fmt.Sprint("Entering Passive Mode (", passPortAddress, ")."))
			case "TYPE":
				if FTPConn.IsAuthenticated() == false {
					FTPConn.sendResponseToClient("530", "Not logged in")
					break
				}
				sendType := command[5:]
				FTPConn.TransferType = sendType
				FTPConn.sendResponseToClient("200", "Set type successful!")
			case "SIZE":
				if FTPConn.IsAuthenticated() == false {
					FTPConn.sendResponseToClient("530", "Not logged in")
					break
				}
				path := command[5:]
				size, err := FTPConn.FileSystem.GetFileSize(path)
				if err != nil {
					FTPConn.sendResponseToClient("550", "Could not get file size")
					break
				}
				Logger.Log(size)
				FTPConn.sendResponseToClient("213", fmt.Sprint(" ", size))
			case "STAT":
				if FTPConn.IsAuthenticated() == false {
					FTPConn.sendResponseToClient("530", "Not logged in")
					break
				}
				path := command[5:]
				stat, err := FTPConn.FileSystem.STAT(path)
				if err != nil {
					Logger.Log("STAT error: ", err)
					FTPConn.sendResponseToClient("550", "Couldn't get STAT")
				}
				FTPConn.sendResponseToClient("213", "-Status")
				FTPConn.sendResponseToClient(stat, "")
				FTPConn.sendResponseToClient("213", " End of status")
			case "PBSZ":
				FTPConn.sendResponseToClient("200", "OK")
			case "PROT":
				FTPConn.sendResponseToClient("200", "OK")
			case "AUTH":
				if len(command) <= 5 {
					FTPConn.sendResponseToClient("500", "No protocol type specified")
				}
				Authtype := command[5:]
				Logger.Log("Client asks for protection using: ", Authtype)
				switch strings.ToUpper(Authtype) {
				case "TLS":
					fallthrough
				case "SSL":
					FTPConn.sendResponseToClient("234", "")
					if err := FTPConn.InitTLSConnection(); err != nil {
						FTPConn.sendResponseToClient("500", "Couldn't use TLS...")
						Logger.Log("TLS error: ", err)
						break
					}
					FTPConn.UsingTLS = true
				}
			case "RNFR":
				if FTPConn.IsAuthenticated() == false {
					FTPConn.sendResponseToClient("530", "Not logged in")
					break
				}
				oldName := command[5:]
				renameobj, err := FTPConn.FileSystem.NewRenameableObj(oldName)
				if err != nil {
					FTPConn.sendResponseToClient("550", "Can't rename obj")
					Logger.Log("Rename object error: ", err)
					break
				}
				FTPConn.actionBuffer.RenameObj = renameobj
				FTPConn.sendResponseToClient("350", "Waiting for RNTO")
			case "RNTO":
				if FTPConn.IsAuthenticated() == false {
					FTPConn.sendResponseToClient("530", "Not logged in")
					break
				}
				newName := command[5:]
				if FTPConn.actionBuffer.RenameObj == nil {
					FTPConn.sendResponseToClient("550", "No RNFR command executed")
					break
				}
				FTPConn.actionBuffer.RenameObj.NewName = newName
				err := FTPConn.FileSystem.Rename(FTPConn.actionBuffer.RenameObj)
				if err != nil {
					FTPConn.sendResponseToClient("550", "Couldn't rename object")
					Logger.Log("RNTO error: ", err)
					break
				}
				FTPConn.sendResponseToClient("250", "Object renamed")
			case "STOR":
				if FTPConn.IsAuthenticated() == false {
					FTPConn.sendResponseToClient("530", "Not logged in")
					break
				}
				fileName := command[5:]
				file, err := FTPConn.FileSystem.STOR(fileName)
				if err != nil {
					FTPConn.sendResponseToClient("550", "Can't create new specified file")
					Logger.Log("STOR error: ", err)
					break
				}
				FTPConn.sendResponseToClient("150", "Ready to receive data")
				err = FTPConn.DataConnection.ReceiveBinaryFile(file.Name())
				if err != nil {
					Logger.Log("STOR error (receiving data): ", err)
					FTPConn.sendResponseToClient("550", "Can't write specified data")
					break
				}
				FTPConn.sendResponseToClient("226", "File transfer complete")
			case "MFMT":
				FTPConn.sendResponseToClient("500", "Not implemented")
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
					FTPConn.sendResponseToClient("430", "Wrong username")
					break
				}
				FTPConn.User = user
				FTPConn.sendResponseToClient("331", "")
				break
			case "PASS":
				pswd := command[5:]
				if FTPConn.User == nil {
					FTPConn.sendResponseToClient("430", "Wrong username")
					break
				}
				if FTPConn.User.CheckPswd(pswd) == false {
					FTPConn.sendResponseToClient("430", "Wrong password")
					FTPConn.User = nil
					break
				}
				//CBModule.RegUser(FTPConn.User.UserName, FTPConn.TCPConn.RemoteAddr().String(), time.Now())
				FTPConn.FileSystem.InitFileSystem(FTPConn.GlobalConfig, FTPConn.User)
				FTPConn.sendResponseToClient("230", "Authenticated")
				//new pass
				break
			case "PORT":
				Logger.Log("PORT sent to Server")
				if FTPConn.IsAuthenticated() == false {
					FTPConn.sendResponseToClient("530", "Not logged in")
					break
				}
				port := command[5:]
				err := FTPConn.DataConnection.InitActiveConnection(port)
				if err != nil {
					FTPConn.sendResponseToClient("550", fmt.Sprint("Dialing active port error: ", err))
					break
				}
				FTPConn.sendResponseToClient("200", fmt.Sprint("PORT command done", FTPConn.DataConnection.FTPActiveDataConnection.DataPortAddress.String()))
			case "RETR":
				if FTPConn.IsAuthenticated() == false {
					FTPConn.sendResponseToClient("530", "Not logged in")
					return
				}
				fileName := command[5:]
				file, err := FTPConn.FileSystem.RETR(fileName)
				if err != nil {
					Logger.Log("RETR Command, fsRETR error: ", err)
					FTPConn.sendResponseToClient("550", "File transfer error")
					return
				}
				FTPConn.sendResponseToClient("150", fmt.Sprint("Opening binary stream for", fileName))
				go func() {
					err = FTPConn.DataConnection.TransferBinaryFile(file)
					if err != nil {
						Logger.Log("RETR command error: ", err)
						FTPConn.sendResponseToClient("550", "File transfer error")
						return
					}
					FTPConn.sendResponseToClient("226", "Transfer complete")
				}()
			case "SYST":
				if FTPConn.IsAuthenticated() == false {
					FTPConn.sendResponseToClient("530", "Not logged in")
					break
				}
				FTPConn.sendResponseToClient("215", runtime.GOOS)
			case "ABOR":
				if FTPConn.IsAuthenticated() == false {
					FTPConn.sendResponseToClient("530", "Not logged in")
					break
				}
				if FTPConn.DataConnection.DataConnectionsClosed() {
					FTPConn.sendResponseToClient("226", "Transfer completed. Data Conn closed")
				} else {
					FTPConn.sendResponseToClient("225", "Data conn opened. Trying to abort data transfer")
					FTPConn.DataConnection.DataTranserAbort = true
				}
			case "QUIT":
				if FTPConn.IsAuthenticated() == false {
					FTPConn.sendResponseToClient("530", "Not logged in")
					break
				}
				Logger.Log("Closing connection")
				FTPConn.CloseConnection(true)
				return
			}
		}
	}
}
