package main

import (
	T "./tcenter"
	S "./ssl"
	"crypto/tls"
	"github.com/op/go-logging"
	"log"
	"os"
	"runtime"
	"syscall"
	"time"
	"flag"
	"os/exec"
)

var connectString = "192.168.1.30:25"
var groupID = "palo"

func daemon(nochdir, noclose int) int {
	var ret, ret2 uintptr
	var err syscall.Errno

	darwin := runtime.GOOS == "darwin"

	// already a daemon
	if syscall.Getppid() == 1 {
		return 0
	}

	// fork off the parent process
	ret, ret2, err = syscall.RawSyscall(syscall.SYS_FORK, 0, 0, 0)
	if err != 0 {
		return -1
	}

	// failure
	if ret2 < 0 {
		os.Exit(-1)
	}

	// handle exception for darwin
	if darwin && ret2 == 1 {
		ret = 0
	}

	// if we got a good PID, then we call exit the parent process.
	if ret > 0 {
		os.Exit(0)
	}

	/* Change the file mode mask */
	_ = syscall.Umask(0)

	// create a new SID for the child process
	s_ret, s_errno := syscall.Setsid()
	if s_errno != nil {
		log.Printf("Error: syscall.Setsid errno: %d", s_errno)
	}
	if s_ret < 0 {
		return -1
	}

	if nochdir == 0 {
		os.Chdir("/")
	}

	if noclose == 0 {
		f, e := os.OpenFile("/dev/null", os.O_RDWR, 0)
		if e == nil {
			fd := f.Fd()
			syscall.Dup2(int(fd), int(os.Stdin.Fd()))
			syscall.Dup2(int(fd), int(os.Stdout.Fd()))
			syscall.Dup2(int(fd), int(os.Stderr.Fd()))
		}
	}

	return 0
}

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
	println("begin 09.15.2")

	// 用户参数
	var d = flag.Bool("d", false, "daemon")
	flag.Parse()

	if !*d {
		println("parent")
		if err := daemon(1, 1); err != 0 {
			println("daemon error", err)
			os.Exit(-1)
		}

		for {
			println("create child")
			child := exec.Command(os.Args[0], "-d")
			err := child.Run()
			if err != nil {
				println(err.Error())
			} else {
				println("child end")
			}

			time.Sleep(time.Duration(1)*time.Minute)
		}
	} else {
		println("child")

		// init log config
		format := logging.MustStringFormatter(`%{message}`)
		backend2 := logging.NewLogBackend(os.Stderr, "", 0)
		backend2Formatter := logging.NewBackendFormatter(backend2, format)
		logging.SetBackend(backend2Formatter)

		connect(connectString, groupID)
	}
}
