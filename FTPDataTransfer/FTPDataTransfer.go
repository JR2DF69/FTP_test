package FTPDataTransfer

import (
	"FTPServ/FTPServConfig"
	"FTPServ/FTPtls"
	"FTPServ/Logger"
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/cheggaaa/pb"
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
	DataTranserAbort         bool
	UsingTLS                 bool
	TLSConfig                *FTPtls.FTPTLSServerParameters
}

type ftpPassiveDataConnection struct {
	DataPortAddress net.TCPAddr
	Listener        net.Listener
	UsingTLS        bool
	TLSConfig       *FTPtls.FTPTLSServerParameters
}
type ftpActiveDataConnection struct {
	DataPortAddress net.TCPAddr
	Connection      *net.TCPConn
	Writer          *bufio.Writer
	Reader          *bufio.Reader
	UsingTLS        bool
	TLSConfig       *FTPtls.FTPTLSServerParameters
}
type DataConnectionMode uint

const (
	DataConnectionModeActive  DataConnectionMode = iota
	DataConnectionModePassive                    = iota
)

func (d *FTPDataConnection) DataConnectionsClosed() bool {
	return d.FTPPassiveDataConnection == nil && d.FTPActiveDataConnection == nil
}
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
	ftppassconn, err := d.initPassiveConnection(pportaddr)
	if err != nil {
		return "", err
	}
	d.FTPPassiveDataConnection = ftppassconn
	d.FTPPassiveDataConnection.UsingTLS = d.UsingTLS
	d.FTPPassiveDataConnection.TLSConfig = d.TLSConfig
	d.dataConnectionMode = DataConnectionModePassive
	Logger.Log(fmt.Sprint("(DataConn *FTPDataConnection) Init(PASSIVE) PASV ADDRESS: ", pportaddr))
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
	lstn, err := net.Listen("tcp", p.DataPortAddress.String())
	if p.UsingTLS {
		lstn = tls.NewListener(lstn, p.TLSConfig.TLSConfig)
	}
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
		return nil
	}
	return nil
}
func (d *FTPDataConnection) GetBinaryFile() error {
	return nil
}
func (d *FTPDataConnection) ReceiveBinaryFile(fileName string) error {
	if err := d.CheckIfConnectionOpened(); err != nil {
		return err
	}
	defer d.CloseConnection()
	if d.dataConnectionMode == DataConnectionModeActive {
	} else if d.dataConnectionMode == DataConnectionModePassive {
		conn, err := d.FTPPassiveDataConnection.Listener.Accept()
		if err != nil {
			return err
		}
		defer conn.Close()
		d.receiveBinaryData(fileName, conn)
	}
	return nil
}
func (d *FTPDataConnection) TransferBinaryFile(file *os.File) error {
	if err := d.CheckIfConnectionOpened(); err != nil {
		return err
	}
	defer d.CloseConnection()
	if d.dataConnectionMode == DataConnectionModeActive {
		d.transferBinaryDataToConnection(file, d.FTPActiveDataConnection.Connection)
		return nil
	} else if d.dataConnectionMode == DataConnectionModePassive {
		conn, err := d.FTPPassiveDataConnection.Listener.Accept()
		defer conn.Close()
		if err != nil {
			return err
		}
		d.transferBinaryDataToConnection(file, conn)
		return nil
	}
	return nil
}
func (d *FTPDataConnection) CheckIfConnectionOpened() error {
	if d.dataConnectionMode == DataConnectionModeActive {
		if d.FTPActiveDataConnection == nil {
			return errors.New("No FTP Data connection (mode:active)")
		}
		if d.FTPActiveDataConnection.Connection == nil {
			return errors.New("TransferBinaryData: Connection not opened (mode:active)")
		}
	} else if d.dataConnectionMode == DataConnectionModePassive {
		if d.FTPPassiveDataConnection == nil {
			return errors.New("No FTP Data connection (mode:passive)")
		}
		if d.FTPPassiveDataConnection.Listener == nil {
			return errors.New("TransferBinaryData: Listener not opened (mode:passive)")
		}
	}
	return nil
}
func (d *FTPDataConnection) transferBinaryDataToConnection(file *os.File, conn net.Conn) {
	sendFileBuff := make([]byte, d.GlobalConfig.BufferSize)
	stats, _ := file.Stat()
	size := stats.Size()
	progressbar := pb.StartNew(int(size))
	progress := 0
	for {
		if d.DataTranserAbort {
			d.DataTranserAbort = false
			Logger.Log("Data transfer aborted")
			progressbar.Finish()
			return
		}
		count, err := file.Read(sendFileBuff)
		progress += count
		progressbar.Set(progress)
		if err == io.EOF {
			progressbar.Finish()
			Logger.Log("Data transfer completed, total ", progress, " bytes")
			break
		}
		conn.Write(sendFileBuff)
	}
}
func (d *FTPDataConnection) receiveBinaryData(fileName string, conn net.Conn) error {
	receiveBuffer := make([]byte, d.GlobalConfig.BufferSize)
	Logger.Log("Receiving data from ", conn.RemoteAddr().String(), "...")
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		Logger.Log("Can't open source file for edit: ", err)
		return err
	}
	writer := bufio.NewWriter(file)
	received := 0
	for {
		if d.DataTranserAbort {
			d.DataTranserAbort = false
			Logger.Log("Data transfer aborted")
			return nil
		}
		rec, err := conn.Read(receiveBuffer)
		if err == nil {
			writer.Write(receiveBuffer[:rec])
			received += rec
			fmt.Printf("\rReceiving data, received %d bytes", received)
		} else {
			err := writer.Flush()
			if err != nil {
				Logger.Log("Data receiving error: ", err)
				file.Close()
				return err
			}
			fmt.Printf("\r\n")
			Logger.Log("Data received, total ", received, " bytes")
			return nil
		}
	}
}
