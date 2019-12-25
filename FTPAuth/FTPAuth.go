package FTPAuth

import (
	"crypto"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
)

const usersFileName string = "users.json"
const hash crypto.Hash = crypto.SHA256

type Users struct {
	Users []User
}

type User struct {
	UserName string
	Password string
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
	json.Unmarshal(users, &Users)
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
