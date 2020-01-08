package FTPtls

import (
	"crypto/tls"
	"os"
)

const serverkeyfilename string = "server.key"
const serverpemfilename string = "server.pem"

type FTPTLSServerParameters struct {
	TLSConfig   *tls.Config
	Certificate tls.Certificate
}

func ReadNewTLSConfig() (*FTPTLSServerParameters, error) {
	cert, err := tls.LoadX509KeyPair(serverpemfilename, serverkeyfilename)
	if err != nil {
		return nil, err
	}
	conf := tls.Config{Certificates: []tls.Certificate{cert}, NextProtos: []string{"ftp"}}
	params := new(FTPTLSServerParameters)
	params.TLSConfig = &conf
	params.Certificate = cert
	return params, nil
}

func readAllSpecifiedFile(filename string) ([]byte, error) {
	File, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	FileStats, err := File.Stat()
	if err != nil {
		return nil, err
	}
	size := FileStats.Size()
	FileBuff := make([]byte, size)
	_, err = File.Read(FileBuff)
	if err != nil {
		return nil, err
	}
	return FileBuff, nil
}
