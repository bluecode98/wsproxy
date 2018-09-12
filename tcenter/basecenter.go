package tcenter

import (
	"crypto/tls"
	"net"
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"time"
	"github.com/op/go-logging"
	"errors"
)

// 普通通讯数据头
type headMessage struct {
	Type	int	   `json:"type,int"`
	Sender  string `json:"sender,omitempty"`
	Target  string `json:"target,omitempty"`
}

// 文件通讯数据头
type fileMessage struct {
	Type		int	   `json:"type,int"`
	Sender  	string `json:"sender,omitempty"`
	Target  	string `json:"target,omitempty"`
	Filename  	string `json:"filename,omitempty"`
}

// 版本信息数据头
type versionMessage struct {
	Version		string	`json:"version"`
	Id  		string 	`json:"ID"`
	Type  		int 	`json:"type"`
	Time  		string 	`json:"time"`
}

// 应答信息数据头
type answerMessage struct {
	Code  		int 	`json:"code"`
	Message  	string 	`json:"msg"`
}

var DefaultVersion = "6.1.0902.1"

// DefaultCenter
var DefaultCenter = &BaseCenter{}

// nilCenter
var nilCenter BaseCenter = *DefaultCenter
var nilLogger *logging.Logger = logging.MustGetLogger("tcenter")

type BaseCenter struct{
	// SSL config
	TLSClientConfig *tls.Config

	// SSL socket
	Connect *tls.Conn

	// clientUID
	ClientUID string

	// targetUID
	TargetUID string

	log *logging.Logger
	connectAddress string
}

//字节转换成整形
func BytesToInt(b []byte) int {
	bytesBuffer := bytes.NewBuffer(b)
	var tmp int32
	binary.Read(bytesBuffer, binary.LittleEndian, &tmp)
	return int(tmp)
}

func cloneTLSConfig(cfg *tls.Config) *tls.Config {
	if cfg == nil {
		return &tls.Config{}
	}
	return cfg.Clone()
}

func (d *BaseCenter) Dial(address string) (error) {
	if d == nil {
		d = &nilCenter
	}

	if d.log == nil {
		d.log = nilLogger
	}

	raddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return err
	}
	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		return err
	}

	// 连接wscenter
	cfg := cloneTLSConfig(d.TLSClientConfig)
	d.Connect = tls.Client(conn, cfg)
	d.connectAddress = address

	// 读取返回的ID信息
	message, _, err := d.RecvMessage()
	if err != nil {
		return err
	}

	d.ClientUID = message.Sender
	d.log.Debug("client id", d.ClientUID)

	// Live report thread
	d.liveReport()

	return nil
}

func (d *BaseCenter) liveReport()  {
	go func() {
		// live report
		liveMessage := &headMessage{
			Type: 100,
		}

		for {
			println("live report", liveMessage, d.ClientUID)
			if err := d.SendMessage(liveMessage, nil); err != nil {
				println(err.Error())
				break
			}

			// sleep times
			time.Sleep(time.Duration(30) * time.Second)
		}
		//println("exit live report")
	}()
}

func (d *BaseCenter) RecvBinMessage() ([]byte, []byte, error) {
	var headBuffer []byte
	var dataBuffer []byte

	buffer := make([]byte, 1024)
	reader := bufio.NewReader(d.Connect)

	// Get head size
	readSize, err := reader.Read(buffer[:4])
	if err != nil {
		return nil, nil, err
	} else if readSize != 4 {
		return nil, nil, errors.New("error size")
	}
	headSize := BytesToInt(buffer)

	// Get data size
	readSize, err = reader.Read(buffer[:4])
	if err!=nil {
		return nil, nil, err
	} else if readSize != 4 {
		return nil, nil, errors.New("error size")
	}
	dataSize := BytesToInt(buffer)
	//d.log.Debug("head size", headSize, "data size", dataSize)

	// Get head
	headBuffer = make([]byte, headSize)
	readSize, err = reader.Read(headBuffer)
	if err!=nil {
		return nil, nil, err
	} else if readSize != headSize {
		return nil, nil, errors.New("error size")
	}

	// Get data
	if dataSize > 0 {
		//d.log.Debug("create data buffer", dataSize)
		dataBuffer = make([]byte, dataSize)
		//d.log.Debug("read...")
		beginPos := 0
		for leftSize := dataSize; leftSize > 0; leftSize -= readSize{
			readSize, err = reader.Read(dataBuffer[beginPos:])
			//d.log.Debug("read", readSize, "left", leftSize)
			if err != nil {
				return nil, nil, err
			}
			beginPos += readSize
		}
	}

	return headBuffer, dataBuffer, nil
}

func (d *BaseCenter) RecvMessage() (*headMessage, []byte, error)  {
	headJSON, data, err := d.RecvBinMessage()
	if err != nil {
		return nil, nil, err
	}

	// change to json object
	header := &headMessage{}
	err = json.Unmarshal(headJSON, header)

	return header, data, err
}

func (d *BaseCenter) sendBinMessage(head []byte, data []byte) (error) {
	// write head & data size
	headSize := int32(len(head))
	dataSize := int32(len(data))
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.LittleEndian, headSize)
	binary.Write(bytesBuffer, binary.LittleEndian, dataSize)
	if _, err := d.Connect.Write(bytesBuffer.Bytes()); err != nil {
		return err
	}

	// write head
	if _, err := d.Connect.Write(head); err != nil {
		return err
	}

	// write data
	if dataSize > 0 {
		if _, err := d.Connect.Write(data); err != nil {
			return err
		}
	}

	return nil
}

func (d *BaseCenter) SendMessage(head *headMessage, data []byte) (error) {
	if head.Sender == "" {
		head.Sender = d.ClientUID
	}
	headJSON, _ := json.Marshal(head)

	return d.sendBinMessage([]byte(headJSON), data)
}

func (d *BaseCenter) sendFileMessage(head *fileMessage, data []byte) (error) {
	if head.Sender == "" {
		head.Sender = d.ClientUID
	}
	headJSON, _ := json.Marshal(head)

	return d.sendBinMessage([]byte(headJSON), data)
}

func (d *BaseCenter) SendExit(data []byte) (error) {
	message := &headMessage{
		Type:	10,
		Target: d.TargetUID,
	}
	return d.SendMessage(message, data)
}
