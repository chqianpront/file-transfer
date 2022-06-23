package client

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"time"

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
)

var targetAddr string
var md5Ch chan models.LxFileInfo = make(chan models.LxFileInfo)
var conn net.Conn

func StartClient(envType string) {
	conf := getConf(envType)
	defer close(md5Ch)
	targetAddr = fmt.Sprintf("%s:%d", conf.Client.Source.Ip, conf.Client.Source.Port)
	var err error
	conn, err = net.DialTimeout("tcp", targetAddr, time.Second*30)
	if err != nil {
		fmt.Printf("connect failed, err : %v\n", err.Error())
		return
	}
	defer conn.Close()
	targetLocation := conf.Client.Target.Location
	handleDir(targetLocation)
	consumeFile()
}

func getConf(env string) *models.ClientConf {
	configFilePath := path.Join(configPath, configFilePrefix+env+configFilePostfix)
	configData, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		fmt.Printf("error: %v\n", err.Error())
	}
	clientConf := new(models.ClientConf)
	yaml.Unmarshal(configData, clientConf)
	return clientConf
}
func handleDir(dirPath string) {
	dir, err := os.Open(dirPath)
	if err != nil {
		fmt.Printf("error: %v\n", err.Error())
		return
	}
	fileInfo, err := dir.Stat()
	if err != nil {
		fmt.Printf("error: %v\n", err.Error())
		return
	}
	if fileInfo.IsDir() {
		lxDirInfo := models.LxDirInfo{
			Name: fileInfo.Name(),
			Path: dirPath,
		}
		cmd := models.Command{
			Type:    cmdDir,
			DirInfo: lxDirInfo,
		}
		str, err := json.Marshal(cmd)
		if err != nil {
			fmt.Printf("error: %s\n", err.Error())
		}
		sendData(str)
		readData()
		files, err := ioutil.ReadDir(dirPath)
		if err != nil {
			fmt.Printf("error: %s\n", err.Error())
			return
		}
		for _, f := range files {
			fmt.Printf("file name : %s\n", f.Name())
			childFilePath := filepath.Join(dirPath, f.Name())
			if f.IsDir() {
				handleDir(childFilePath)
			} else {
				go handleFile(childFilePath)
			}
		}
	} else {
		go handleFile(dirPath)
	}
}
func handleFile(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("error: %v\n", err.Error())
		return
	}
	fileStat, err := file.Stat()
	if err != nil {
		fmt.Printf("error: %v\n", err.Error())
		return
	}
	h := md5.New()
	io.Copy(h, file)
	lxFileInfo := models.LxFileInfo{
		Name:     file.Name(),
		Path:     filePath,
		Md5:      hex.EncodeToString(h.Sum(nil)),
		FileSize: fileStat.Size(),
	}
	if lxFileInfo.FileSize <= 0 {
		return
	}
	md5Ch <- lxFileInfo
}
func consumeFile() {
	for {
		lxFileInfo := <-md5Ch
		cmd := models.Command{
			Type:     cmdFile,
			FileInfo: lxFileInfo,
		}
		str, err := json.Marshal(cmd)
		if err != nil {
			fmt.Printf("error: %s\n", err.Error())
		}
		sendData(str)
		bres := readData()
		res := string(bres)
		switch res {
		case "OK":
			transFile(lxFileInfo.Path)
			readData()
		case "PASS":
			fmt.Printf("%s dont need transfer\n", lxFileInfo.Name)
		case "ERR":
			fmt.Printf("%s send file error\n", lxFileInfo.Name)
		}
	}
}
func transFile(filePath string) {
	file, err := os.Open(filePath)
	buf := make([]byte, 65542)
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
	}
	fs, _ := file.Stat()
	fileSize := fs.Size()
	for {
		rlen, err := file.Read(buf)
		if rlen < 65542 {
			buf = buf[0:rlen]
		}
		fileSize = fileSize - int64(rlen)
		conn.Write(buf)
		if fileSize <= 0 {
			break
		}
		if err == io.EOF {
			break
		}
	}
}

func sendData(data []byte) {
	fmt.Printf("send to client %s\n", data)

	l := len(data)
	magicNum := make([]byte, 4)
	binary.BigEndian.PutUint32(magicNum, 0x123456)
	lenNum := make([]byte, 2)
	binary.BigEndian.PutUint16(lenNum, uint16(l))
	packetBuf := bytes.NewBuffer(magicNum)
	packetBuf.Write(lenNum)
	packetBuf.Write(data)
	_, err := conn.Write(packetBuf.Bytes())
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

func readData() []byte {
	var res []byte
	result := bytes.NewBuffer(nil)
	var buf [65542]byte // 由于 标识数据包长度 的只有两个字节 故数据包最大为 2^16+4(魔数)+2(长度标识)
	n, err := conn.Read(buf[0:])
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
