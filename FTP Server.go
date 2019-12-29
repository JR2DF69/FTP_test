// FTP Server
package main

import (
	"FTPServ/FTPAuth"
	"FTPServ/FTPServConfig"
	"FTPServ/FTPServer"
	"FTPServ/Logger"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		showHelp()
		return
	}
	config, err := FTPServConfig.ReadConfig()
	if err != nil {
		Logger.Log("func main(): couldn't load server configuration. Run server with -rs key to reset config.json")
		return
	}
	commandWithArgs := strings.Split(args[0], " ")
	command := commandWithArgs[0]
	switch command {
	case "-sp":
		value, err := strconv.Atoi(commandWithArgs[1])
		if err != nil {
			fmt.Println(err)
			return
		}
		if err != nil {
			fmt.Println(err)
			return
		}
		err = config.SetPort(value)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Port set to: ", config.Config.Port)
	case "-pp":
		port1, err := strconv.Atoi(commandWithArgs[1])
		port2, err2 := strconv.Atoi(commandWithArgs[2])
		if (err != nil) || (err2 != nil) {
			fmt.Println("Wrong portnums to set dataport range!")
			showHelp()
			return
		}
		if port2 < port1 {
			port1, port2 = port2, port1
		}
		if err != nil {
			fmt.Println(err)
			return
		}
		err = config.SetDataPort(port1, port2)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Passive data port set to: ", config.Config.DataPortLow, "-", config.Config.DataPortHigh)
	case "-an":
		value, err := strconv.ParseBool(commandWithArgs[1])
		if err != nil {
			fmt.Println(err)
			return
		}
		config.SetAnonymous(value)
		fmt.Println("Anonymous set to: ", config.Config.Anonymous)
	case "-wd":
		value := commandWithArgs[1]
		if err = config.SetHomeDir(value); err != nil {
			fmt.Println("Setting home dir error: ", err)
		}
		fmt.Println("FTP Homedir changed to: ", value)
	case "-rs":
		FTPServConfig.CreateConfig()
		fmt.Println("Loaded default server configuration")
		return
	case "-bs":
		value, err := strconv.Atoi(commandWithArgs[1])
		if err != nil {
			fmt.Println("Couldn't set buffer size: ", err)
			return
		}
		config.SetBufferSize(value)
		fmt.Println("New buffersize value: ", config.Config.BufferSize)
	case "-pd":
		config.Print()
		return
	case "-start":
		Logger.Log("Starting server>")
		if err != nil {
			Logger.Log("func main(): couldn't load server configuration. Run configurator to repair config.json")
			return
		}
		users, err := FTPAuth.LoadUsersList()
		if err != nil {
			Logger.Log("func main(): failed to load users configuration. Server stops now(", err, ")")
			return
		}
		FTPServer.StartFTPServer(config.Config, users)
	default:
		showHelp()
		return
	}
	config.SaveConfig()
}
func showHelp() {
	fmt.Println("PN FTP Server Configurator commands:\r\n'-sp port_num' - set message port\r\n'-pp port_numlow-port_numhigh' - set passive mode data port range\r\n'-wd path_to_dir' - set root directory\r\n'-an (true|false) || (0|1) - set anonymous user allowed\r\n'-mp' - set num of max peers\r\n'-rs' - reset config to default\r\n'-pd' - prints config file")
	fmt.Println("'-bs size' - set send and receive buffer size (bytes)")
	fmt.Println("PN FTP Server users commands: \r\nUnder construction")
	fmt.Println("Run with -start to run FTP server")
}
