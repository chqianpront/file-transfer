package server

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"

	"chen.com/file-trans/models"
	"gopkg.in/yaml.v3"
)

const (
	configPath        = "conf"
	configFilePrefix  = "application-"
	configFilePostfix = ".yml"
	cmdDir            = 1
	cmdFile           = 2
	cmdErr            = -1
	svrOk             = "OK"
	svrErr            = "ERR"
	svrPass           = "PASS"
)

type ServerConf struct {
	Name   string
	Server struct {
		Port     int
		Ip       string
		Location string
	}
}

var conf *ServerConf
var destPrefix string

func StartServer(envType string) {
	conf = getConf(envType)
	destPrefix = conf.Server.Location
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
		fmt.Printf("error : %s\n", err)
	}
	serverConf := new(ServerConf)
	yaml.Unmarshal(configData, serverConf)
	return serverConf
}

func handleConn(lc net.Conn) {
	defer lc.Close()
	defer fmt.Printf("connection close address %s\n", lc.RemoteAddr())
	fmt.Printf("accpet remote address %s\n", lc.RemoteAddr())
	for {
		buf := readFromLc(lc)
		buf = bytes.Trim(buf, "\x00")
		if len(buf) <= 0 {
			continue
		}
		cmd := new(models.Command)
		// yaml.Unmarshal(buf, cmd)
		json.Unmarshal(buf, cmd)
		// fmt.Printf("cmd is %s\n", string(buf))
		switch cmd.Type {
		case cmdDir:
			fmt.Printf("dir command\n")
			lxDirInfo := cmd.DirInfo
			destDir := savePath(lxDirInfo.Path)
			os.MkdirAll(destDir, os.ModePerm)
			fmt.Printf("dir: %s created\n", destDir)
			sendToClient(lc, svrOk)
		case cmdFile:
			fmt.Printf("file command\n")
			lxFileInfo := cmd.FileInfo
			destLocation := path.Join(conf.Server.Location, lxFileInfo.Path)
			destLocation = savePath(destLocation)
			_, err := os.Stat(destLocation)
			if err != nil {
				sendToClient(lc, svrOk)
				recvFile(lc, lxFileInfo)
				sendToClient(lc, svrOk)
			} else {
				sendToClient(lc, svrPass)
			}
		default:
			sendToClient(lc, svrErr)
		}
	}
}
func recvFile(lc net.Conn, lxFileInfo models.LxFileInfo) {
	buf := make([]byte, 65542)
	filePath := lxFileInfo.Path
	fileSize := lxFileInfo.FileSize
	filePath = savePath(filePath)
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
	}
	for {
		rlen, err := lc.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("error: %s\n", err.Error())
		}
		if rlen <= 65542 {
			buf = buf[0:rlen]
		}
		fileSize = fileSize - int64(rlen)
		file.Write(buf)
		if fileSize <= 0 {
			break
		}
	}
	fmt.Printf("file writed: %s\n", filePath)
}
func savePath(sPath string) string {
	return path.Join(destPrefix, sPath)
}
func sendToClient(lc net.Conn, str string) {
	fmt.Printf("send to client %s\n", str)
	data := []byte(str)

	l := len(data)
	magicNum := make([]byte, 4)
	binary.BigEndian.PutUint32(magicNum, 0x123456)
	lenNum := make([]byte, 2)
	binary.BigEndian.PutUint16(lenNum, uint16(l))
	packetBuf := bytes.NewBuffer(magicNum)
	packetBuf.Write(lenNum)
	packetBuf.Write(data)
	_, err := lc.Write(packetBuf.Bytes())
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
	}
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

func readFromLc(lc net.Conn) []byte {
	var res []byte
	result := bytes.NewBuffer(nil)
	var buf [65542]byte // 由于 标识数据包长度 的只有两个字节 故数据包最大为 2^16+4(魔数)+2(长度标识)
	n, err := lc.Read(buf[0:])
	result.Write(buf[0:n])
	if err != nil {
		if err == io.EOF {
		} else {
			fmt.Println("read err:", err)
		}
	} else {
		scanner := bufio.NewScanner(result)
		scanner.Split(packetSlitFunc)
		for scanner.Scan() {
			fmt.Println("recv:", string(scanner.Bytes()[6:]))
			res = scanner.Bytes()[6:]
			break
		}
	}
	result.Reset()
	return res
}
