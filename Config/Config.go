// Config
package Config

type ConfigStorage struct {
	Port          int
	Anonymous     bool
	FTPRootFolder string
	DataPort      int
	BufferSize    int
}

func LoadConfig() (config ConfigStorage) {
	config.Port = 4011
	config.Anonymous = false
	config.FTPRootFolder = "/Users/panovnm"
	config.DataPort = 20000
	config.BufferSize = 32068
	return config
}
