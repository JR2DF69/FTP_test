// FTP Server
package main

import (
	"FTPServ/FTPAuth"
	"FTPServ/FTPServConfig"
	"FTPServ/FTPServer"
	"FTPServ/Logger"
)

func main() {
	Logger.Log("Starting server>")
	var err error
	config, err := FTPServConfig.LoadConfig()
	if err != nil {
		Logger.Log("func main(): couldn't load server configuration. Run configurator to repair config.json")
		return
	}
	users, err := FTPAuth.LoadUsersList()
	if err != nil {
		Logger.Log("func main(): failed to load users configuration. Server stops now(", err, ")")
		return
	}
	FTPServer.StartFTPServer(config, users)
}
