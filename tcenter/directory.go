package tcenter

import (
	"fmt"
	"os"
	"encoding/csv"
	"encoding/json"
	"github.com/kataras/iris/core/errors"
	"io/ioutil"
	"path"
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

	if message.Type == 101 {
		//targetUID := message.Sender
		//version := &versionMessage{}
		//if err := json.Unmarshal(data, version); err != nil {
		//	return err
		//}
		hostInfo := &HostInfo{}
		if err := json.Unmarshal(data, hostInfo); err != nil {
			return err
		}
		hostInfo.IP2 = message.Remote		// 更新上线IP
		hostInfo.Live = true
		hostInfo.Error = 5					// 错误记数，小于0表示已经超时没有心跳

		// 检查信息文件
		infoFilename := path.Join("data", hostInfo.Group, hostInfo.Id, "info.json")
		if _, err := os.Stat(infoFilename); err != nil {
			os.MkdirAll(path.Join("data", hostInfo.Group, hostInfo.Id), os.ModePerm)
		}

		// 写入文件
		if infoData, err := json.Marshal(hostInfo); err == nil {
			ioutil.WriteFile(infoFilename, infoData, 0666)
		}

		d.log.Debug(hostInfo.Id, "live")
		//d.log.Debug("********************************************************")
		//d.log.Debug("version:", hostInfo.Version)
		//d.log.Debug("id:", hostInfo.Id)
		//d.log.Debug("uid:", hostInfo.Uid)
		//d.log.Debug("type:", hostInfo.Type)
		//d.log.Debug("time:", hostInfo.Time)
		//d.log.Debug("name:", hostInfo.Name)
		//d.log.Debug("system:", hostInfo.System)
		//d.log.Debug("memo:", hostInfo.Memo)
		//d.log.Debug("group:", hostInfo.Group)
		//d.log.Debug("local ip:", hostInfo.IP1)
		//d.log.Debug("remote ip:", hostInfo.IP2)
	} else if message.Type == 104 {
		//clientList := make(map[string]interface{})
		clientList := make(map[string]interface{})
		json.Unmarshal(data, &clientList)
		if len(clientList) == 0 {
			d.log.Debug("not find live client")
		} else {
			//d.log.Debug("find live client")
			for k, v := range clientList {
				targetUID := k
				serverId := v.(string)

				d.log.Debug("query", serverId)
				queryMessage := &headMessage{
					Type:		101,
					Target:		targetUID,
				}
				d.SendMessage(queryMessage, nil)
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

func (d *DirectoryCenter) GetListMap(groupID string) (map[string]HostInfo, error) {
	infoPath := "data"
	hostList := make(map[string]HostInfo)

	dir, err := ioutil.ReadDir(infoPath)
	if err != nil {
		return nil, err
	}

	for _, fi := range dir {
		if !fi.IsDir() {
			continue
		}
		//serverUID := k
		serverID := fi.Name()
		println(serverID)

		info := HostInfo{
			Id:  serverID,
			//Uid: serverUID,
		}

		systemInfo, _ := d.checkServerInfo(serverID)
		if len(systemInfo)>0 {
			info.Name = systemInfo[0]
			info.System = systemInfo[1] + systemInfo[2]
			info.Memo = systemInfo[13]
		}
		hostList[serverID] = info
	}

	return hostList, nil
}

// 直接获取信息
func (d *DirectoryCenter) GetListMap2(groupID string) (map[string]HostInfo, error) {
	listMessage := &headMessage{
		Type: 104,
	}
	d.SendMessage(listMessage, []byte(groupID))

	message, data, err := d.RecvMessage()
	if err != nil {
		return nil, err
	}

	if message.Type == 104 {
		hostList := make(map[string]HostInfo)
		clientList := make(map[string]interface{})
		json.Unmarshal(data, &clientList)
		if len(clientList) == 0 {
			return nil, nil
		} else {
			//d.log.Debug("find live client")
			//hostList := make([]HostInfo, 5)
			//println(hostList)
			for k, v := range clientList {
				serverUID := k
				serverID := v.(string)

				info := HostInfo{
					Id:  serverID,
					Uid: serverUID,
				}

				systemInfo, _ := d.checkServerInfo(serverID)
				if len(systemInfo)>0 {
					info.Name = systemInfo[0]
					info.System = systemInfo[1] + systemInfo[2]
					info.Memo = systemInfo[13]
					//info.IP1 =
				}
				hostList[serverID] = info
			}
		}

		return hostList, nil
	} else {
		return nil, errors.New("error message")
	}
}

