package models

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

type LxFileInfo struct {
	Name     string
	Path     string
	FileSize int64
	Md5      string
}

type LxDirInfo struct {
	Name string
	Path string
}

type Command struct {
	Type     int
	FileInfo LxFileInfo
	DirInfo  LxDirInfo
}
