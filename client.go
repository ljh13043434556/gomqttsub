package main

import (
	"fmt"

	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
)

const theme = "lizonggezigui/accept"                                               //订阅主题
const mqtt_server = "123.207.127.201"                                              //mqtt服务器地址
const mqtt_client_id = "beck_accept"                                               //mqtt客户端ID
const mqtt_user = "beck"                                                           //mqtt帐号
const mqtt_pass = "123456"                                                         //mqtt密码
const php_server = "http://lzsmgzg.app.xiaozhuschool.com/wxsite/device_notify/api" //接收数据的php接口

var c mqtt.Client                  //mqtt 客户端
var urlCh = make(chan string)      //入接收到的内容放入这个通道 中
var taskCh = make(chan string, 50) //把进行中的POST操作放入这个通道中，用于控制并发数，100就是最多100个POST操作同时进行

func main() {
	if os.Getppid() != 1 {
		//判断当其是否是子进程，当父进程return之后，子进程会被 系统1 号进程接管
		filePath, _ := filepath.Abs(os.Args[0])
		//将命令行参数中执行文件路径转换成可用路径
		cmd := exec.Command(filePath)
		//将其他命令传入生成出的进程
		cmd.Stdin = os.Stdin
		//给新进程设置文件描述符，可以重定向到文件中
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		//开始执行新进程，不等待新进程退出
		cmd.Start()
		return
	}

	//初始化客户端，没有链接的话一直链接，直到链接成功为为止
	initMqttClient()

	go loop_() //运行 处理通道数据的程序 （把数据提交到PHP后台）

	for {
		//无限循环,
		time.Sleep(time.Duration(2) * time.Second)
	}
	fmt.Println("断开链接")
	c.Disconnect(250)

}

//初始化链接
func initMqttClient() {
	for {
		if mqttConnect() {
			return
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
}

//订阅主题
func subscribe() bool {
	//定义，接收到数据后的回调函数
	var f mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
		var content = msg.Payload()
		var payload string = ""
		//把BYTE数据转成字符串
		for _, value := range content {
			payload += string(value)
		}
		fmt.Println("get", payload)
		urlCh <- payload
	}
	//订阅主题
	if token := c.Subscribe(theme, 0, f); token.Wait() && token.Error() != nil {
		fmt.Println("订阅失败", token.Error()) //订阅失败
		return false
	} else {
		//订阅成功
		fmt.Println("订阅成功") //订阅失败
		return true

	}

}

//链接MQQT服务器
func mqttConnect() bool {
	opts := mqtt.NewClientOptions().AddBroker("tcp://" + mqtt_server + ":1883").SetClientID(mqtt_client_id)
	opts.SetKeepAlive(time.Duration(30) * time.Second) // 定时发送数据，保持链接
	opts.SetAutoReconnect(true)                        //这个好像是自动链接
	opts.SetMaxReconnectInterval(time.Duration(1) * time.Second)
	opts.SetUsername(mqtt_user)
	opts.SetPassword(mqtt_pass)

	//链接断开后的事件
	var lostf mqtt.ConnectionLostHandler = func(c mqtt.Client, err_ error) {
		subscribe()
	}
	opts.SetConnectionLostHandler(lostf)

	c = mqtt.NewClient(opts) //创建一个客户端类
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println("链接失败", token.Error())
		return false

	} else {
		//订阅主题
		return subscribe()
	}

}

//从通道中拿，接收的数据，进行处理
func loop_() {
	fmt.Println("loop function")

	for {
		var data string = <-urlCh

		//控制并发
		taskCh <- data

		go httpPost(php_server, data)
	}
}

//http post操作
func httpPost(url_ string, data string) {
	_, err := http.PostForm(url_,
		url.Values{"data": {data}})
	if err != nil {
		fmt.Println("post error: " + data)
	}
	fmt.Println("post ok: " + data)

	var _ string = <-taskCh
}

//进行http get操作
func GetData(url string) {

	client := &http.Client{}
	resp, err := client.Get(url)
	defer resp.Body.Close()
	if err != nil {
		fmt.Println("http error", err)
	} else {
		fmt.Println("http ok", url)
	}

	var _ string = <-taskCh

}
