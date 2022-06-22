package client

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net"
	"path"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	configPath        = "conf"
	configFilePrefix  = "application-"
	configFilePostfix = ".yml"
)

type ClientConf struct {
	Name   string
	Client struct {
		Source struct {
			Port int
			Ip   string
		}
		Target struct {
			Location string
		}
	}
}

func StartClient(envType string) {
	conf := getConf(envType)
	targetAddr := fmt.Sprintf("%s:%d", conf.Client.Source.Ip, conf.Client.Source.Port)
	data := []byte("[这里才是一个完整的数据包]")
	l := len(data)
	fmt.Println(l)
	magicNum := make([]byte, 4)
	binary.BigEndian.PutUint32(magicNum, 0x123456)
	lenNum := make([]byte, 2)
	binary.BigEndian.PutUint16(lenNum, uint16(l))
	packetBuf := bytes.NewBuffer(magicNum)
	packetBuf.Write(lenNum)
	packetBuf.Write(data)
	conn, err := net.DialTimeout("tcp", targetAddr, time.Second*30)
	if err != nil {
		fmt.Printf("connect failed, err : %v\n", err.Error())
		return
	}
	for i := 0; i < 1000; i++ {
		_, err = conn.Write(packetBuf.Bytes())
		if err != nil {
			fmt.Printf("write failed , err : %v\n", err)
			break
		}
	}
}

func getConf(env string) *ClientConf {
	configFilePath := path.Join(configPath, configFilePrefix+env+configFilePostfix)
	configData, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		fmt.Printf("error : %s", err)
	}
	clientConf := new(ClientConf)
	yaml.Unmarshal(configData, clientConf)
	return clientConf
}
