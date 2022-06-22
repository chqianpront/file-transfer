package server

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
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
	for {
		fmt.Println("waiting for connection")
		lc, err := l.Accept()
		if err != nil {
			fmt.Printf("error: %s\n", err)
			continue
		}
		go handleConn(lc)
	}
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
func packetSlitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// 检查 atEOF 参数 和 数据包头部的四个字节是否 为 0x123456(我们定义的协议的魔数)
	if !atEOF && len(data) > 6 && binary.BigEndian.Uint32(data[:4]) == 0x123456 {
		var l int16
		// 读出 数据包中 实际数据 的长度(大小为 0 ~ 2^16)
		binary.Read(bytes.NewReader(data[4:6]), binary.BigEndian, &l)
		pl := int(l) + 6
		if pl <= len(data) {
			return pl, data[:pl], nil
		}
	}
	return
}

func handleConn(lc net.Conn) {
	defer lc.Close()
	fmt.Printf("accpet remote address %s\n", lc.RemoteAddr())
	result := bytes.NewBuffer(nil)
	var buf [65542]byte
	for {
		blen, err := lc.Read(buf[0:])
		result.Write(buf[0:blen])
		if err != nil {
			if err == io.EOF {
				continue
			} else {
				fmt.Printf("error: %s\n", err)
				break
			}
		} else {
			scanner := bufio.NewScanner(result)
			scanner.Split(packetSlitFunc)
			for scanner.Scan() {
				ret := string(scanner.Bytes()[6:])
				fmt.Printf("message received is %s\n", ret)
			}
		}
		result.Reset()
	}
}
