package main

import (
	T "./tcenter"
	S "./ssl"
	"crypto/tls"
	"github.com/op/go-logging"
	"os"
	"time"
)

func connect(address string, group string) error {
	// 客户端
	wsserver := &T.WSServer{}

	// init ssl
	cert, _ := S.ClientCert()
	wsserver.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}

	// connect wscenter
	err := wsserver.Dial(address)
	if err != nil {
		//log.Debug(err.Error())
		return err
	}

	// 绑定到指定分组
	wsserver.Bind(group)

	// 等待命令
	wsserver.Listen()

	return nil
}

func main() {
	// init log config
	format := logging.MustStringFormatter(`%{message}`)
	backend2 := logging.NewLogBackend(os.Stderr, "", 0)
	backend2Formatter := logging.NewBackendFormatter(backend2, format)
	logging.SetBackend(backend2Formatter)

	for {
		connect("173.230.150.215:25", "test")
		println("wait...")
		time.Sleep(time.Duration(3)*time.Second)
	}
}
