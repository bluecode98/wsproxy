package tcenter

import (
	"io/ioutil"
	"errors"
	"encoding/json"
	"path/filepath"
	"os"
	"fmt"
)

type WSFile struct {
	BaseCenter
	targetID string
}

// WSFile
func (d *WSFile) recvThread() {
	go func() {
		for {
			head, data, err := d.RecvBinMessage()
			if err != nil {
				return
			}

			message := &headMessage{}
			err = json.Unmarshal(head, message)
			//d.log.Debug("recv", message.Type)

			if message.Type == 106 {
				//line := strings.Trim(string(data), "\n")
				//d.log.Debug(line)
				d.log.Debug(string(data))
			} else if message.Type == 203 {
				fileInfo := &fileMessage{}
				json.Unmarshal(head, fileInfo)
				//saveFilename := fileInfo.Filename
				_, absFilename := filepath.Split(fileInfo.Filename)
				//d.log.Debug(paths, absFilename)
				saveFilename := fmt.Sprintf(".\\data\\%s\\%s", d.targetID, absFilename)
				//d.log.Debug(absFilename, saveFilename)

				file, err := os.OpenFile(saveFilename, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
				if err != nil {
					d.log.Debug(err.Error())
					continue
				}
				_, err = file.Write(data)
				file.Close()
				d.log.Debug("save to", saveFilename)
			} else {
				d.log.Debug("unkown type", message.Type)
			}
		}
	}()
}

func (d *WSFile) Connect(targetID string) (error) {
	//d.log.Debug("query target shell")
	d.targetID = targetID
	message := &headMessage{
		Type:	103,
		Target: d.targetID,
	}
	d.SendMessage(message, []byte("file"))

	message, data, err := d.RecvMessage()
	if err != nil {
		return err
	}
	if message.Type == 101 {
		d.TargetUID = message.Sender
	} else {
		println("target error")
		return errors.New("target error")
	}
	d.log.Debug("connect ok")
	d.log.Debug(string(data))

	// 启动数据监听
	//d.recvThread()

	return nil
}

// shell 命令
func (d *WSFile) SendShell(shell string) (error) {
	message := &headMessage{
		Type:	201,
		Target: d.TargetUID,
	}
	d.SendMessage(message, []byte(shell))

	return nil
}

// 上传文件
func (d *WSFile) SendUploadFile(localName string, remoteName string) (error) {
	fileData, err := ioutil.ReadFile(localName)
	if err != nil {
		return err
	}

	// 文件消息
	message := &fileMessage{
		Type:		106,
		Target: 	d.TargetUID,
		Filename: 	remoteName,
	}
	d.sendFileMessage(message, fileData)

	head, data, err := d.RecvBinMessage()
	if err != nil {
		return err
	}

	// 解析数据
	answer := &answerMessage{}
	if err := json.Unmarshal(head, message); err != nil {
		return err
	}
	if err := json.Unmarshal(data, answer); err != nil {
		return err
	}

	if message.Type != 106 {
		return errors.New("error type")
	}
	if answer.Code > 0 {
		return errors.New(answer.Message)
	}

	return nil
}

// 下载文件
func (d *WSFile) SendDownloadFile(remoteName string) (error) {
	// 文件消息
	message := &fileMessage{
		Type:		202,
		Target: 	d.TargetUID,
		Filename: 	remoteName,
	}
	d.sendFileMessage(message, nil)

	return nil
}