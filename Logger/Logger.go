// Logger
package Logger

import (
	"fmt"
	"net"
	"time"
)

type LogType uint

const (
	SimpleMessage   LogType = iota
	CriticalMessage         = iota
	UserAction              = iota
)

type LoggerConfig struct {
	ConnID   uint
	ConnPort net.Addr
}

//Log writes to CommandLine and SysLogger Server messages
//args are the arguments for log
func (lc *LoggerConfig) Log(logtype LogType, args ...interface{}) {
	fmt.Println("ConnID: ", lc.ConnID, " on port ", lc.ConnPort.String(), " : ", args)
}
func NewLogger(connID uint, connPort net.Addr) *LoggerConfig {
	lgc := new(LoggerConfig)
	lgc.ConnID = connID
	lgc.ConnPort = connPort
	return lgc
}
func Log(args ...interface{}) {
	timeNow := time.Now()
	fmt.Println(timeNow, " : ", args)
}
