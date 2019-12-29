package FTPDataTransfer

import (
	"FTPServ/FTPServConfig"
	"FTPServ/Logger"
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
)

var config *FTPServConfig.ConfigStorage

type FTPDataConnection struct {
	dataConnectionMode       DataConnectionMode
	FTPPassiveDataConnection *ftpPassiveDataConnection
	FTPActiveDataConnection  *ftpActiveDataConnection
	TCPServerAddress         string
	GlobalConfig             *FTPServConfig.ConfigStorage
}

type ftpPassiveDataConnection struct {
	DataPortAddress net.TCPAddr
	Listener        *net.TCPListener
}
type ftpActiveDataConnection struct {
	DataPortAddress net.TCPAddr
	Connection      *net.TCPConn
	Writer          *bufio.Writer
	Reader          *bufio.Reader
}
type DataConnectionMode uint

const (
	DataConnectionModeActive  DataConnectionMode = iota
	DataConnectionModePassive                    = iota
)

func NewConnection(serveraddr string, servconf *FTPServConfig.ConfigStorage) (*FTPDataConnection, error) {
	if serveraddr == "" || servconf == nil {
		return nil, errors.New("NewDataConnection: wrong parameters")
	}
	dc := new(FTPDataConnection)
	dc.TCPServerAddress = serveraddr
	dc.GlobalConfig = servconf
	return dc, nil
}
func (d *FTPDataConnection) CloseConnection() error {
	if d.FTPPassiveDataConnection != nil {
		if d.FTPPassiveDataConnection.Listener != nil {
			err := d.FTPPassiveDataConnection.Listener.Close()
			if err != nil {
				return err
			}
			d.FTPPassiveDataConnection.Listener = nil
			d.FTPPassiveDataConnection = nil
		}
	}
	//close active connection
	if d.FTPActiveDataConnection != nil {
		if d.FTPActiveDataConnection.Connection != nil {
			err := d.FTPActiveDataConnection.Connection.Close()
			if err != nil {
				return err
			}
			d.FTPActiveDataConnection.Connection = nil
			d.FTPActiveDataConnection = nil
		}
	}
	return nil
}

//для ответа клиенту
func (d *FTPDataConnection) GetDataPortAddress() (string, error) {
	ipAddress := d.TCPServerAddress
	ipAddressSplitted := strings.Split(ipAddress, ".")
	dataPortString, err := d.countPassiveConnDataPort()
	if err != nil {
		return "", err
	}
	ipAddressSplitted = append(ipAddressSplitted, dataPortString[0], dataPortString[1])
	ipAddressPort := strings.Join(ipAddressSplitted, ",")
	return ipAddressPort, nil
}

//для жизни
func (d *FTPDataConnection) countPassiveConnDataPort() ([]string, error) {
	portnum := -1
	for i := d.GlobalConfig.DataPortLow; i <= d.GlobalConfig.DataPortHigh; i++ {
		port, err := net.Listen("tcp", fmt.Sprint(d.TCPServerAddress, ":", i))
		if err == nil {
			port.Close()
			portnum = i
			break
		}
	}
	if portnum == -1 {
		return nil, errors.New("No free dataports...")
	}
	part2 := math.Mod(float64(portnum), 256)
	part1 := (float64(portnum) - part2) / 256
	dataPortString := []string{strconv.Itoa(int(part1)), strconv.Itoa(int(part2))}
	return dataPortString, nil
}
func (d *FTPDataConnection) InitPassiveConnection() (string, error) {
	if d.FTPPassiveDataConnection != nil {
		if d.FTPPassiveDataConnection.Listener != nil {
			err := d.CloseConnection()
			if err != nil {
				return "", err
			}
		}
	}
	pportaddr, err := d.GetDataPortAddress()
	if err != nil {
		return "", err
	}
	Logger.Log(fmt.Sprint("(DataConn *FTPDataConnection) Init(PASSIVE) PASV ADDRESS: ", pportaddr))
	ftppassconn, err := d.initPassiveConnection(pportaddr)
	if err != nil {
		return "", err
	}
	d.FTPPassiveDataConnection = ftppassconn
	d.dataConnectionMode = DataConnectionModePassive
	return pportaddr, ftppassconn.openConnection()
}
func (d *FTPDataConnection) InitActiveConnection(clientaddr string) error {
	if d.FTPActiveDataConnection != nil {
		if d.FTPActiveDataConnection.Connection != nil {
			err := d.CloseConnection()
			if err != nil {
				return err
			}
		}
	}
	aportaddr, err := d.parseDataPortAddr(clientaddr)
	if err != nil {
		return err
	}
	conn, err := net.DialTCP("tcp", nil, &aportaddr)
	if err != nil {
		return err
	}
	ActiveConn := new(ftpActiveDataConnection)
	ActiveConn.DataPortAddress = aportaddr
	ActiveConn.Connection = conn
	ActiveConn.Reader = bufio.NewReader(conn)
	ActiveConn.Writer = bufio.NewWriter(conn)
	d.FTPActiveDataConnection = ActiveConn
	d.dataConnectionMode = DataConnectionModeActive
	return nil
}
func (p *ftpPassiveDataConnection) openConnection() error {
	fmt.Println(&p.DataPortAddress)
	lstn, err := net.ListenTCP("tcp", &p.DataPortAddress)
	if err != nil {
		return err
	}
	p.Listener = lstn
	return nil
}
func (d *FTPDataConnection) initPassiveConnection(DataPort string) (*ftpPassiveDataConnection, error) {
	tcpaddr, err := d.parseDataPortAddr(DataPort)
	if err != nil {
		return nil, err
	}
	PassConn := new(ftpPassiveDataConnection)
	PassConn.DataPortAddress = tcpaddr
	return PassConn, nil
}

func (d *FTPDataConnection) parseDataPortAddr(dataPort string) (net.TCPAddr, error) {
	PortParamsSplitted := strings.Split(dataPort, ",")
	num1, _ := strconv.Atoi(PortParamsSplitted[4])
	num2, _ := strconv.Atoi(PortParamsSplitted[5])
	portnum := num1*256 + num2
	ip := net.ParseIP(fmt.Sprint(PortParamsSplitted[0], ".", PortParamsSplitted[1], ".", PortParamsSplitted[2], ".", PortParamsSplitted[3]))
	tcpaddr := net.TCPAddr{ip, portnum, ""}
	return tcpaddr, nil
}
func (d *FTPDataConnection) TransferASCIIData(data string) error {
	if d.dataConnectionMode == DataConnectionModePassive {
		if d.FTPPassiveDataConnection == nil {
			return errors.New("No passive TCP connection found for client! Type PASV to run passive mode connection")
		}
		if d.FTPPassiveDataConnection.Listener == nil {
			return errors.New("No passive TCP listener found for client! Type PASV to run passive mode connection")
		}
		dataConn, err := d.FTPPassiveDataConnection.Listener.Accept()
		if err != nil {
			return err
		}
		writer := bufio.NewWriter(dataConn)
		writer.Write([]byte(data))
		writer.Write([]byte{13, 10})
		writer.Flush()
		dataConn.Close()
		return d.CloseConnection()
	}
	if d.dataConnectionMode == DataConnectionModeActive {
		if d.FTPActiveDataConnection == nil {
			return errors.New("No active TCP connection found for server. Type PORT (h1,h2,h3,h4,h5,h6) to run active mode connection")
		}
		if d.FTPActiveDataConnection.Connection == nil {
			return errors.New("No active TCP connection found for server. Type PORT (h1,h2,h3,h4,h5,h6) to run active mode connection")
		}
		d.FTPActiveDataConnection.Writer.Write([]byte(data))
		d.FTPActiveDataConnection.Writer.Write([]byte{13, 10})
		d.FTPActiveDataConnection.Writer.Flush()
		return d.CloseConnection()
	}
	return nil
}
func (d *FTPDataConnection) GetBinaryFile() error {
	return nil
}
func (d *FTPDataConnection) TransferBinaryFile(file *os.File) error {
	if d.dataConnectionMode == DataConnectionModeActive {
		if d.FTPActiveDataConnection == nil {
			return errors.New("No FTP Data connection (mode:active)")
		}
		if d.FTPActiveDataConnection.Connection == nil {
			return errors.New("TransferBinaryData: Connection not opened (mode:active)")
		}
		d.transferBinaryDataToConnection(file, d.FTPActiveDataConnection.Connection)
		Logger.Log("Active connection closed")
		return d.CloseConnection()
	} else if d.dataConnectionMode == DataConnectionModePassive {
		if d.FTPPassiveDataConnection == nil {
			return errors.New("No FTP Data connection (mode:passive)")
		}
		if d.FTPPassiveDataConnection.Listener == nil {
			return errors.New("TransferBinaryData: Listener not opened (mode:passive)")
		}
		conn, err := d.FTPPassiveDataConnection.Listener.Accept()
		if err != nil {
			return err
		}
		d.transferBinaryDataToConnection(file, conn)
		conn.Close()
		Logger.Log("Passive connection closed")
		return d.CloseConnection()
	}

	return nil
}
func (d *FTPDataConnection) transferBinaryDataToConnection(file *os.File, conn net.Conn) {
	sendFileBuff := make([]byte, d.GlobalConfig.BufferSize)
	for {
		_, err := file.Read(sendFileBuff)
		if err == io.EOF {
			break
		}
		conn.Write(sendFileBuff)
	}
}
