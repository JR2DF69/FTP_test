package FTPAuth

import (
	"crypto"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

const usersFileName string = "users.json"
const hash crypto.Hash = crypto.SHA256

type Users struct {
	Users     []User
	usersFile *os.File
}

type User struct {
	UserName string
	Password string
	Folder   string
}

func LoadUsersList() (*Users, error) {
	Users := new(Users)
	usersFile, err := os.Open(usersFileName)
	if err != nil {
		return nil, err
	}
	users, err := ioutil.ReadAll(usersFile)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(users, &Users.Users)
	Users.usersFile = usersFile
	return Users, nil
}
func HashPswd(pswd *string) {
	hash := sha256.New()
	*pswd = hex.EncodeToString(hash.Sum([]byte(*pswd)))
}
func (U *User) CheckPswd(pswd string) bool {
	HashPswd(&pswd)
	if strings.Compare(pswd, U.Password) == 0 {
		return true
	}
	return false
}
func (U *Users) CheckUserName(userName string) *User {
	for _, usr := range U.Users {
		if userName == usr.UserName {
			return &usr
		}
	}
	return nil
}
func (U *Users) AddNewUser(userName, password, folder string) error {
	if (len(strings.TrimSpace(userName)) == 0) || (len(strings.TrimSpace(password)) == 0) {
		return errors.New("Wrong User name or password sent")
	}
	HashPswd(&password)
	user := User{UserName: userName, Password: password, Folder: folder}
	U.Users = append(U.Users, user)
	return nil
}
func (U *Users) Save() error {
	if U.usersFile == nil {
		file, err := os.Create(usersFileName)
		if err != nil {
			return errors.New(fmt.Sprint("func SaveConfig() error: ", err))
		}
		U.usersFile = file
	}
	output, err := json.Marshal(U.Users)
	if err != nil {
		return errors.New(fmt.Sprint("func SaveUsers() error: ", err))
	}
	fmt.Println(U.usersFile.Name())
	ioutil.WriteFile(U.usersFile.Name(), output, os.ModeAppend)
	U.usersFile.Close()
	return err
}
func (U *Users) RemoveUser(user *User) error {
	usrIndex := -1
	for i, usr := range U.Users {
		if usr.UserName == user.UserName {
			usrIndex = i
		}
	}
	if usrIndex == -1 {
		return errors.New("No such user specified")
	}
	newUsrSlice := append(U.Users[:usrIndex], U.Users[usrIndex+1:]...)
	U.Users = newUsrSlice
	return nil
}
