// Config
package Config

type ConfigStorage struct {
	Port          int
	Anonymous     bool
	FTPRootFolder string
	DataPort      int
}

func LoadConfig() (config ConfigStorage) {
	config.Port = 4010
	config.Anonymous = false
	config.FTPRootFolder = "/Users/panovnm"
	config.DataPort = 4017
	return config
}
