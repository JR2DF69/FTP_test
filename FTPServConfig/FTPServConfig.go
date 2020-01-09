// Config
package FTPServConfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type Configurator struct {
	Config     *ConfigStorage
	configFile *os.File
}

const MaxPeer int = 500

type ConfigStorage struct {
	Port           int
	Anonymous      bool
	FTPRootFolder  string
	DataPortLow    int
	DataPortHigh   int
	MaxClientValue int
	BufferSize     int
}

func LoadConfig() (config *ConfigStorage, err error) {
	cfg, err := ReadConfig()
	if err != nil {
		return nil, err
	}
	config = cfg.Config
	config.BufferSize = 1024
	return config, nil
}
func (c *Configurator) Print() {
	fmt.Println("Dataport = ", c.Config.DataPortLow, "-", c.Config.DataPortHigh, "\r\nPort = ", c.Config.Port, "\r\nMax peers = ", c.Config.MaxClientValue, "\r\nAllow anonymous = ", c.Config.Anonymous, "\r\nRoot folder = ", c.Config.FTPRootFolder, "\r\n")
	fmt.Println("BufferSize = ", c.Config.BufferSize)
}
func (c *Configurator) SetAnonymous(value bool) {
	c.Config.Anonymous = value
}
func (c *Configurator) SetBufferSize(newSize int) {
	c.Config.BufferSize = newSize
}
func ReadConfig() (*Configurator, error) {
	file, err := os.Open("config.json")
	if err != nil {
		fmt.Println("Can't read config file: ", err, "\r\nRecreate configuration file? [Y|N]: ")
		var answer string
		fmt.Scan(&answer)
		if strings.ToLower(answer) == "y" {
			cfgrt := CreateConfig()
			return cfgrt, nil
		}
		return nil, errors.New("")
	}
	configData, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, errors.New(fmt.Sprint("Can't read config file: ", err))
	}
	cfg := new(ConfigStorage)
	err = json.Unmarshal(configData, cfg)
	if err != nil {
		fmt.Println("Can't unmarshal config file: ", err, "\r\nConfig file returned to default")
		cfgrt := CreateConfig()
		return cfgrt, nil
	}
	cfgrt := new(Configurator)
	cfgrt.Config = cfg
	cfgrt.configFile = file
	return cfgrt, nil
}

func CreateConfig() *Configurator {
	cfgrt := new(Configurator)
	cfgrt.Config = new(ConfigStorage)
	cfgrt.Config.Port = 21
	cfgrt.Config.Anonymous = false
	cfgrt.Config.MaxClientValue = 100
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/"
	}
	cfgrt.SetHomeDir(homeDir)
	cfgrt.SetDataPort(20, (20 + cfgrt.Config.MaxClientValue))
	cfgrt.SaveConfig()
	return cfgrt
}
func (c *Configurator) SetHomeDir(dir string) error {
	dirStat, err := os.Stat(dir)
	if err != nil {
		return errors.New(fmt.Sprint("func setHomeDir error: ", err))
	}
	if dirStat.IsDir() != true {
		return errors.New(fmt.Sprint("func setHomeDir error: not a directory"))
	}
	c.Config.FTPRootFolder = dir
	return nil
}
func (c *Configurator) SetMaxPeer(maxpeer int) error {
	if maxpeer > MaxPeer {
		return errors.New(fmt.Sprint("Max peer is limited ", MaxPeer, " by server!"))
	}
	if maxpeer <= 0 {
		return errors.New("Peer value must be at least one")
	}
	if c.dataPortValueValid(c.Config.DataPortLow, c.Config.DataPortHigh) == false {
		err := c.SetDataPort(c.Config.DataPortLow, c.Config.DataPortLow+maxpeer)
		if err != nil {
			return errors.New("Couldn't set MaxPeer: check Data port range first")
		}
	}
	c.Config.MaxClientValue = maxpeer
	return nil
}
func (c *Configurator) SetPort(port int) error {
	if c.portValueValid(port) == false {
		return errors.New(fmt.Sprint("func SetPort() error: port value is not valid. Also, check data port range."))
	}
	c.Config.Port = port
	return nil
}
func (c *Configurator) portValueValid(port int) bool {
	return ((port < 65535) && (port > 0) && (port != c.Config.DataPortLow) && (port != c.Config.DataPortHigh) && !(port >= c.Config.DataPortLow && port <= c.Config.DataPortHigh))
}
func (c *Configurator) SetDataPort(portlow, porthigh int) error {
	if c.dataPortValueValid(portlow, porthigh) == false {
		return errors.New("func SetDataPort() error: port value is not valid")
	}
	c.Config.DataPortLow = portlow
	c.Config.DataPortHigh = porthigh
	return nil
}
func (c *Configurator) dataPortValueValid(portlow, porthigh int) bool {
	return ((portlow < 65535) && (portlow > 0) && (portlow != c.Config.Port)) && ((porthigh < 65535) && (porthigh > 0) && (porthigh != c.Config.Port) && (portlow <= (porthigh - c.Config.MaxClientValue)) && !(c.Config.Port >= portlow && c.Config.Port <= porthigh))
}
func (c *Configurator) SaveConfig() error {
	if c.configFile == nil {
		file, err := os.Create("config.json")
		if err != nil {
			return errors.New(fmt.Sprint("func SaveConfig() error: ", err))
		}
		c.configFile = file
	}
	output, err := json.Marshal(c.Config)
	if err != nil {
		return errors.New(fmt.Sprint("func SaveConfig() error: ", err))
	}
	ioutil.WriteFile(c.configFile.Name(), output, os.ModeAppend)
	c.configFile.Close()
	return err
}
