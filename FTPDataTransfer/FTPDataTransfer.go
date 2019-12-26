package FTPDataTransfer

import (
	"FTPServ/FTPServConfig"
	"FTPServ/Logger"
	"bufio"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
)

var config *FTPServConfig.ConfigStorage

type FTPDataConnection struct {
	DataConnectionMode       DataConnectionMode
	ActiveDataConnection     *net.TCPConn
	FTPPassiveDataConnection *ftpPassiveDataConnection
	Writer                   *bufio.Writer
	Reader                   *bufio.Reader
	TCPServerAddress         string
	GlobalConfig             *FTPServConfig.ConfigStorage
}

type ftpPassiveDataConnection struct {
	DataPortAddress net.TCPAddr
	Listener        net.Listener
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
		port, err := net.Listen("tcp", fmt.Sprint(":", i))
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

/*func (DataConn *FTPDataConnection) Init(ConnectionMode DataConnectionMode, DataPort string) error {
	DataConn.DataConnectionMode = ConnectionMode
	if ConnectionMode == DataConnectionModeActive {
		DataConn.parseDataPortAddr(DataPort)
		Logger.Log(fmt.Sprint("(DataConn *FTPDataConnection) Init(ACTIVE) ACTV ADDRESS: ", DataConn.DataPortAddress))
		return nil
	} else if ConnectionMode == DataConnectionModePassive {

		return nil
	}
	return nil
}*/
func (d *FTPDataConnection) InitPassiveConnection() (string, error) {
	pportaddr, err := d.GetDataPortAddress()
	if err != nil {
		return "", err
	}
	Logger.Log(fmt.Sprint("(DataConn *FTPDataConnection) Init(PASSIVE) PASV ADDRESS: ", pportaddr))
	ftppassconn, err := d.initPassiveConnection(pportaddr)
	if err != nil {
		return "", err
	}
	return pportaddr, ftppassconn.openConnection()
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

/*func (DataConn *FTPDataConnection) OpenConnection() error {
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
func (DataConn *FTPDataConnection) sendBinaryData(dataBytes []byte) error {
	/*if DataConn.CheckConnectionOpened() == false {
		DataConn.OpenConnection()
	}
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
}*/
