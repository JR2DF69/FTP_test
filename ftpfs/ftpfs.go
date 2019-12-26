// ftpfs.go
package ftpfs

import (
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
}

func (fsParams *FileSystem) InitFileSystem(config *FTPServConfig.ConfigStorage) {
	fsParams.FTPRootFolder = config.FTPRootFolder
}

func (fsParams *FileSystem) checkFTPWDForSlash() string {
	if len(fsParams.FTPWorkingDirectory) == 0 {
		return "/"
	}
	firstSym := fsParams.FTPWorkingDirectory[0]
	if firstSym != '/' {
		return fmt.Sprint("/", fsParams.FTPWorkingDirectory)
	}
	return fsParams.FTPWorkingDirectory
}
func (fsParams *FileSystem) LIST(directory string) ([]string, error) {
	if directory == "" {
		//using working directory
		directory = fsParams.FTPWorkingDirectory
	}
	directory = fmt.Sprint(fsParams.FTPRootFolder, fsParams.checkFTPWDForSlash())
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
	directoryForCheck := fmt.Sprint(fsParams.FTPRootFolder, "/", directory)
	err := checkIfDir(directoryForCheck)
	if err != nil {
		return err
	}
	fsParams.FTPWorkingDirectory = directory
	return nil
}
func (fsParams *FileSystem) RETR(fileName string) (*os.File, error) {
	fullFileName := fmt.Sprint(fsParams.FTPRootFolder, fsParams.checkFTPWDForSlash(), fileName)
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
