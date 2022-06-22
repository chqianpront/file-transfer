package server

import (
	"fmt"
	"io/ioutil"
	"net"
	"path"

	"gopkg.in/yaml.v3"
)

const (
	configPath        = "conf"
	configFilePrefix  = "application-"
	configFilePostfix = ".yml"
)

type ServerConf struct {
	Name   string
	Server struct {
		Port     int
		Ip       string
		Location string
	}
}

func StartServer(envType string) {
	conf := getConf(envType)
	addr := fmt.Sprintf("%s:%d", conf.Server.Ip, conf.Server.Port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	go func() {
		for {
			fmt.Println("waiting for connection")
			lc, err := l.Accept()
			if err != nil {
				fmt.Printf("error: %s\n", err)
				continue
			}
			go handleConn(lc)
		}
	}()
}
func getConf(env string) *ServerConf {
	configFilePath := path.Join(configPath, configFilePrefix+env+configFilePostfix)
	configData, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		fmt.Printf("error : %s", err)
	}
	serverConf := new(ServerConf)
	yaml.Unmarshal(configData, serverConf)
	return serverConf
}
func handleConn(lc net.Conn) {
	fmt.Printf("accpet remote address %s", lc.RemoteAddr())
	b := make([]byte, 1024)
	blen, err := lc.Read(b)
	if err != nil {
		fmt.Printf("error: %s\n", err)
	}
	fmt.Printf("byte read len is %d", blen)
}
