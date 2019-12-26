package main

import (
	"FTPServ/FTPServConfig"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var config FTPServConfig.Configurator

func main() {
	config, err := FTPServConfig.ReadConfig()
	if err != nil {
		fmt.Println(err)
		return
	}
	args := os.Args[1:]
	//semantic: arg paramvalue
	//-sp port_num
	//-pp port_num
	//-wd path_to_dir
	//-an true/false 0 1
	//-rs reset to default
	if len(args) == 0 {
		showHelp()
		return
	}
	argsString := strings.Join(args, " ")
	command := argsString[0:3]
	if len(command) < 4 && !(command != "-rs" || command != "-rd") {
		showHelp()
		return
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
		if err != nil {
			fmt.Println(err)
			showHelp()
			return
		}
		if err = config.SetHomeDir(value); err != nil {
			fmt.Println("Setting home dir error: ", err)
		}
	case "-mp":
		value, err := strconv.Atoi(argsString[4:])
		if err != nil {
			fmt.Println(err)
			showHelp()
			return
		}
		err = config.SetMaxPeer(value)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Max FTP client: ", config.Config.MaxClientValue)
	case "-rs":
		FTPServConfig.CreateConfig()
		return
	case "-pd":
		config.Print()
		return
	default:
		showHelp()
		return
	}
	if err = config.SaveConfig(); err != nil {
		fmt.Println(err)
		return
	}
}
func showHelp() {
	fmt.Println("PN FTP Server Configurator commands:\r\n'-sp port_num' - set message port\r\n'-pp port_numlow-port_numhigh' - set passive mode data port range\r\n'-wd path_to_dir' - set root directory\r\n'-an (true|false) || (0|1) - set anonymous user allowed\r\n'-mp' - set num of max peers\r\n'-rs' - reset config to default\r\n'-pd' - prints config file")
}
