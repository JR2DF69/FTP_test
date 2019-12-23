// ftpfs.go
package ftpfs

import (
	"FTPServ/Config"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type FileSystem struct {
	FTPRootFolder       string
	FTPWorkingDirectory string
}

func (fsParams *FileSystem) InitFileSystem(config *Config.ConfigStorage) {
	fsParams.FTPRootFolder = config.FTPRootFolder
}

func (fsParams *FileSystem) checkFTPWDForSlash() string {
	if len(fsParams.FTPWorkingDirectory) == 0 {
		return fsParams.FTPWorkingDirectory
	}
	firstSym := fsParams.FTPWorkingDirectory[0]
	if firstSym != '/' {
		return fmt.Sprint("/", fsParams.FTPWorkingDirectory)
	}
	return fsParams.FTPWorkingDirectory
}
func (fsParams *FileSystem) LIST(directory string) []string {
	if directory == "" {
		//using working directory
		directory = fsParams.FTPWorkingDirectory
	}
	directory = fmt.Sprint(fsParams.FTPRootFolder, fsParams.checkFTPWDForSlash())
	dirStat, err := os.Stat(directory)
	if dirStat.IsDir() == false {
		return make([]string, 1)
	}
	lsOutput, err := executeShellCommands("ls", "-ltr", directory)
	if err != nil {
		return nil
	}
	lsArray := strings.Split(lsOutput, "\n")
	var outputString string
	for _, line := range lsArray {
		if len(line) >= 4 {
			firstFive := line[:5]
			if strings.ToLower(firstFive) == "total" {
				continue
			}
		} else {
			continue
		}
		outputString = fmt.Sprint(outputString, "\r\n", line)
		fmt.Println(line)
	}
	outputArray := strings.Split(outputString, "\r\n")
	return outputArray
}
func (fsParams *FileSystem) CWD(directory string) error {
	fsParams.FTPWorkingDirectory = directory
	return nil
}

func (fsParams *FileSystem) GetFileSize(FileName string) (size int64, err error) {
	fileName := fmt.Sprint(fsParams.FTPRootFolder, FileName)
	fileInfo, err := os.Stat(fileName)
	if err != nil {
		return 0, err
	}
	if fileInfo.IsDir() {
		return 0, errors.New("Is a folder")
	}
	return fileInfo.Size(), nil
}

func executeShellCommands(command string, args ...string) (string, error) {
	if runtime.GOOS == "windows" {
		return "", errors.New("Can't execute commands of Windows machine")
	}
	output, err := exec.Command(command, args...).Output()
	if err != nil {
		fmt.Println("ERROR: ", err)
		return "", err
	}
	return string(output[:]), nil
}
