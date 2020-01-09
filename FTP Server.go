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
	users, err := FTPAuth.LoadUsersList()
	if err != nil {
		Logger.Log("func main(): failed to load users configuration. Server stops now(", err, ")")
		return
	}
	argsString := strings.Join(args, " ")
	command := argsString[0:3]
	stopServer := make(chan bool)
	if len(command) < 4 && !(command != "-rs" || command != "-rd") {
		showHelp()
		return
	}
	if command == "-st" {
		command = "-start"
	}
	if command == "-ss" {
		command = "-sstart"
	}
	switch command {
	case "-sp":
		value, err := strconv.Atoi(argsString[4:])
		if err != nil {
			fmt.Println(err)
			showHelp()
			return
		}
		err = config.SetPort(value)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Port set to: ", config.Config.Port)
	/*case "-sc":
		value, err := strconv.ParseBool(argsString[4:])
		if err != nil {
			fmt.Println(err)
			showHelp()
			return
		}
		config.SetSecurePortEnabled(value)
		fmt.Println("Secure port set to: ", config.Config.FTPSEnabled)
	case "-sa":
		value, err := strconv.Atoi(argsString[4:])
		if err != nil {
			fmt.Println(err)
			showHelp()
			return
		}
		err = config.SetPort(value)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Port set to: ", config.Config.Port)*/
	case "-pp":
		ports := strings.Split(argsString[4:], "-")
		if len(ports) != 2 {
			fmt.Println("Wrong portrange sent to set")
			showHelp()
			return
		}
		port1, err := strconv.Atoi(ports[0])
		port2, err2 := strconv.Atoi(ports[1])
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
		value, err := strconv.ParseBool(argsString[4:])
		if err != nil {
			fmt.Println(err)
			showHelp()
			return
		}
		config.SetAnonymous(value)
		fmt.Println("Anonymous set to: ", config.Config.Anonymous)
	case "-wd":
		value := argsString[4:]
		if err = config.SetHomeDir(value); err != nil {
			fmt.Println("Setting home dir error: ", err)
		}
		fmt.Println("FTP Homedir changed to: ", value)
	case "-rs":
		FTPServConfig.CreateConfig()
		fmt.Println("Loaded default server configuration")
		return
	case "-bs":
		value, err := strconv.Atoi(argsString[4:])
		if err != nil {
			fmt.Println("Couldn't set buffer size: ", err)
			return
		}
		config.SetBufferSize(value)
		fmt.Println("New buffersize value: ", config.Config.BufferSize)
	case "-pd":
		config.Print()
		return
	case "-adduser":
		userParams := strings.Split(argsString[4:], " ")
		if len(userParams) != 3 {
			fmt.Println("Wrong new user params!")
			showHelp()
			return
		}
		UserName := userParams[0]
		Password := userParams[1]
		Folder := userParams[2]
		user := users.CheckUserName(UserName)
		if user != nil {
			fmt.Println("User already exist on server!")
			return
		}
		if err = users.AddNewUser(UserName, Password, Folder); err != nil {
			fmt.Println("Couldn't add new user: ", err)
			return
		}
		fmt.Println("User ", UserName, " added to server and could log in.")
	case "-rmuser":
		UserName := argsString[4:]
		user := users.CheckUserName(UserName)
		if user == nil {
			fmt.Println("No user with Username ", UserName, " found on server")
			return
		}
		if err = users.RemoveUser(user); err != nil {
			fmt.Print("Remove user error: ", err)
			return
		}
		fmt.Println("User ", UserName, " removed from server")
	case "-prusers":
		for i, usr := range users.Users {
			fmt.Println("User ", i+1, ": User name = ", usr.UserName, ", root folder: ", usr.Folder)
		}
	case "-start":
		Logger.Log("Starting server>")
		go FTPServer.StartFTPServer(config.Config, users, stopServer, false)
		readAfterStart(&stopServer)
	case "-sstart":
		Logger.Log("Starting server (FTPS mode)>")
		go FTPServer.StartFTPServer(config.Config, users, stopServer, true)
		readAfterStart(&stopServer)
	default:
		showHelp()
		return
	}
	config.SaveConfig()
	users.Save()
}

func readAfterStart(stopServer *(chan bool)) {
	readln := ""
	for {
		fmt.Scanln(&readln)
		if strings.ToLower(readln) == "exit" || strings.ToLower(readln) == "stop" {
			fmt.Println("Stopping server...")
			*stopServer <- true
			break
		}
	}
}

func showHelp() {
	fmt.Println("PN FTP Server Configurator commands:\r\n'-sp port_num' - set message port\r\n'-pp port_numlow port_numhigh' - set passive mode data port range\r\n'-wd path_to_dir' - set root directory\r\n'-an (true|false) || (0|1) - set anonymous user allowed\r\n'-mp' - set num of max peers\r\n'-rs' - reset config to default\r\n'-pd' - prints config file")
	fmt.Println("'-bs size' - set send and receive buffer size (bytes)")
	fmt.Println("PN FTP Server users commands: \r\nUnder construction")
	fmt.Println("'-adduser Username Password Folder' - add user with specified name, password and root folder (/ is FTP root folder)")
	fmt.Println("'-rmuser Username' - remove specified user")
	fmt.Println("'-prusers' - prints users list")
	fmt.Println("Run with -start to run FTP server")
	fmt.Println("Run with -sstart to run FTPS server (TLS certificate and key required in root server folder)")
	fmt.Println("'exit' or 'stop' - stops FTP Server")
}
