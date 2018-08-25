package tcenter

import (
	"fmt"
	"os"
	"encoding/csv"
	"encoding/json"
)

type DirectoryCenter struct {
	BaseCenter
}

// DirectoryCenter
func (d *DirectoryCenter) recvThread() (error) {
	//println("recv message")
	message, data, err := d.RecvMessage()
	if err != nil {
		return err
	}

	if message.Type == 104 {
		clientList := make(map[string]interface{})
		json.Unmarshal(data, &clientList)
		if len(clientList) == 0 {
			d.log.Debug("not find live client.")
		} else {
			d.log.Debug("find live client")
			for k, v := range clientList {
				targetUID := k
				serverId := v.(string)
				serverInfo, _ := d.checkServerInfo(serverId)
				if len(serverInfo)==0 {
					// 查询主机信息
					d.log.Debug("query", serverId, "info")
					queryMessage := &headMessage{
						Type:	105,
						Target: targetUID,
					}
					d.SendMessage(queryMessage, nil)
				} else {
					infoDetail := fmt.Sprintf("%s %s %s", serverId, serverInfo[0], serverInfo[1])
					d.log.Debug(infoDetail)
				}
			}
		}
	} else if message.Type == 105 {
		// 保存主机信息
		d.log.Debug("save", message.Sender, "info")
		clientID := message.Sender

		infoFilename := fmt.Sprintf(".\\data\\%s\\systeminfo.csv", clientID)
		file, err := os.OpenFile(infoFilename, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
		file.Write(data)
		file.Close()
	} else {
		d.log.Debug("unkown message", message.Type)
	}

	return nil
}

func (d *DirectoryCenter) checkServerInfo(serverId string) ([]string, error) {
	infoFilename := fmt.Sprintf(".\\data\\%s\\systeminfo.csv", serverId)
	// 判断文件是否存在
	_, err := os.Stat(infoFilename)
	if err != nil {
		os.MkdirAll(".\\data\\" + serverId, os.ModePerm)

		return nil, nil
	} else {
		infoFile, err := os.Open(infoFilename)
		defer infoFile.Close()
		if err != nil {
			return nil, err
		}
		reader := csv.NewReader(infoFile)
		line, _ := reader.ReadAll()
		return line[1], nil
	}
}

func (d *DirectoryCenter) Listen()  {
	go func() {
		for {
			d.recvThread()
		}
	}()
}

func (d *DirectoryCenter) GetList(groupID string) (error) {
	listMessage := &headMessage{
		Type: 104,
	}
	d.SendMessage(listMessage, []byte(groupID))

	return nil
}

