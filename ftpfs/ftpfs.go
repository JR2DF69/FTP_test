// ftpfs.go
package ftpfs

import (
	"FTPServ/FTPAuth"
	"FTPServ/FTPServConfig"
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
	FSUser              *FTPAuth.User
}

func (fsParams *FileSystem) InitFileSystem(config *FTPServConfig.ConfigStorage, user *FTPAuth.User) {
	fsParams.FSUser = user
	fsParams.FTPRootFolder = config.FTPRootFolder
	if user.Folder != "/" {
		fsParams.FTPRootFolder = fmt.Sprint(fsParams.FTPRootFolder, user.Folder)
	}
	lastCharIndex := len(fsParams.FTPRootFolder)
	lastChar := fsParams.FTPRootFolder[lastCharIndex-1]
	if lastChar == '/' {
		fsParams.FTPRootFolder = fsParams.FTPRootFolder[:len(fsParams.FTPRootFolder)-1]
	}
}

func (fsParams *FileSystem) checkForSlash(checking string) string {
	if len(checking) == 0 {
		return "/"
	}
	firstSym := checking[0]
	if firstSym != '/' {
		return fmt.Sprint("/", checking)
	}
	return checking
}
func (fsParams *FileSystem) removeFirstSlash(checking string) string {
	if len(checking) == 0 {
		return checking
	}
	firstsym := checking[0]
	if firstsym == '/' {
		return checking[1:]
	}
	return checking
}
func (fsParams *FileSystem) LIST(directory string) ([]string, error) {
	if directory == "" {
		//using working directory
		directory = fsParams.FTPWorkingDirectory
	}
	directory = fmt.Sprint(fsParams.FTPRootFolder, fsParams.checkForSlash(fsParams.FTPWorkingDirectory))
	err := checkIfDir(directory)
	if err != nil {
		return nil, err
	}
	lsOutput, err := executeShellCommands("ls", "-ltr", directory)
	if err != nil {
		return nil, err
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
	return outputArray, nil
}
func checkIfDir(dirName string) error {
	dirStat, err := os.Stat(dirName)
	if err != nil {
		return err
	}
	if dirStat.IsDir() == false {
		return errors.New("Not a dir")
	}
	return nil
}
func (fsParams *FileSystem) CWD(directory string) error {
	directory = fsParams.checkForSlash(directory)
	directoryForCheck := fmt.Sprint(fsParams.FTPRootFolder, directory)
	err := checkIfDir(directoryForCheck)
	if err != nil {
		return err
	}
	fsParams.FTPWorkingDirectory = directory
	return nil
}
func (fsParams *FileSystem) RETR(fileName string) (*os.File, error) {
	workingPath := fmt.Sprint(fsParams.FTPRootFolder, fsParams.checkForSlash(fsParams.FTPWorkingDirectory))
	fullFileName := fmt.Sprint(workingPath, fsParams.removeFirstSlash(fileName))
	fi, err := os.Stat(fullFileName)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, errors.New("RETR File is dir")
	}
	file, err := os.Open(fullFileName)
	if err != nil {
		return nil, err
	}
	return file, nil
}
func (fsParams *FileSystem) GetFileSize(FileName string) (size int64, err error) {
	workingPath := fmt.Sprint(fsParams.FTPRootFolder, fsParams.checkForSlash(fsParams.FTPWorkingDirectory))
	fileName := fmt.Sprint(workingPath, fsParams.removeFirstSlash(FileName))
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
