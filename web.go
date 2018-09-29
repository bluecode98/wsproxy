package main

import (
	T "./tcenter"
	S "./ssl"
	"github.com/kataras/iris"
	"io/ioutil"
	"path"
	"os"
	"encoding/json"
	"time"
	"crypto/tls"
)

type User struct{
	Username    string `json:"username"`
	Password    string `json:"password"`
}

type HostList struct{
	GroupID    string `json:"group"`
}

type UploadFileinfo struct{
	GroupID   	string `json:"group"`
	TargetID	string `json:"targetid"`
	TargetUID	string `json:"targetuid"`
	RemoteName	string `json:"remote"`
}

var connectString = "66.175.221.141:25"
//var connectString = "127.0.0.1:4433"
var groupID = "iron"
const maxSize = 5 << 20 // 5MB

func GetClientFromPath(groupID string) (map[string]T.HostInfo, error) {
	infoPath := path.Join("data", groupID)
	hostList := make(map[string]T.HostInfo)

	dir, err := ioutil.ReadDir(infoPath)
	if err != nil {
		return nil, err
	}

	for _, fi := range dir {
		if !fi.IsDir() {
			continue
		}
		infoFilename := path.Join(infoPath, fi.Name(), "info.json")
		if _, err := os.Stat(infoFilename); err != nil {
			continue
		}

		//println(infoFilename)
		if infoData, err := ioutil.ReadFile(infoFilename); err == nil {
			hostInfo := &T.HostInfo{}
			if err := json.Unmarshal(infoData, hostInfo); err != nil {
				continue
			}

			hostList[hostInfo.Id] = *hostInfo
		}
	}

	return hostList, nil
}

// 在线判断
func CheckIsLive(groupID string)  {
	infoPath := path.Join("data", groupID)
	println("CheckIsLive", infoPath)

	dir, err := ioutil.ReadDir(infoPath)
	if err != nil {
		return
	}

	for _, fi := range dir {
		if !fi.IsDir() {
			continue
		}
		infoFilename := path.Join(infoPath, fi.Name(), "info.json")
		if _, err := os.Stat(infoFilename); err != nil {
			continue
		}

		//println(infoFilename)
		if infoData, err := ioutil.ReadFile(infoFilename); err == nil {
			hostInfo := &T.HostInfo{}
			if err := json.Unmarshal(infoData, hostInfo); err != nil {
				continue
			}

			if hostInfo.Error < 0 {
				hostInfo.Live = false
				println(hostInfo.Name, "down")
			} else {
				hostInfo.Error--
			}

			// 写入文件
			if infoData, err := json.Marshal(hostInfo); err == nil {
				ioutil.WriteFile(infoFilename, infoData, 0666)
			} else {
				println("write", infoFilename, "error")
			}
		}
	}
}

// 查询是否在线
func ListClient(address string, group string)  {
	// 用户列表
	directory := &T.DirectoryCenter{}

	// init ssl
	cert, err := S.ClientCert()
	directory.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}

	// connect wscenter
	err = directory.Dial(address)
	if err != nil {
		println("link Error", err.Error())
		os.Exit(0)
	}
	directory.Listen()

	for {
		println("query", address, group)
		directory.GetList(group)
		time.Sleep(time.Duration(30) * time.Second)
		CheckIsLive(group)
	}
}


func main() {
	//go ListClient(connectString, groupID)

	// create web server
	app := iris.New()

	app.Logger().SetLevel("debug")
	// Optionally, add two built'n handlers
	// that can recover from any http-relative panics
	// and log the requests to the terminal.
	//app.Use(recover.New())
	//app.Use(logger.New())

	// Method:   GET
	// Resource: http://localhost:8080
	//app.Handle("GET", "/", func(ctx iris.Context) {
	//	ctx.HTML("<h1>Welcome</h1>")
	//})
	app.StaticWeb("/", "./html")

	app.Post("/user/login", func(ctx iris.Context) {
		c := &User{}

		if err := ctx.ReadJSON(c); err != nil{
			//panic(err.Error())
			ctx.JSON(iris.Map{
				"code": 1,
				"msg":  err.Error(),
			})
			return
		}

		ctx.JSON(iris.Map{
			"code":  0,
			"data": iris.Map{
				"user": c.Username,
				"group": 3693,
				"login": true,
			},
			"msg": "login success!",
		})
	})

	app.Post("/host/list", func(ctx iris.Context) {
		h := &HostList{}

		if err := ctx.ReadJSON(h); err != nil{
			//panic(err.Error())
			ctx.JSON(iris.Map{
				"code": 1,
				"msg":  err.Error(),
			})
			return
		}

		hostList, err := GetClientFromPath(h.GroupID)
		if err != nil {
			ctx.JSON(iris.Map{
				"code": 1,
				"msg":  err.Error(),
			})
			return
		}

		ctx.JSON(iris.Map{
			"code":  0,
			"data": hostList,
			"msg": "成功获取节点列表",
		})
	})

	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}
