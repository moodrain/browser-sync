package main

import (
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Channel struct {
	Name string
	Content string
	Sockets []*websocket.Conn
}

type ConnWrapper struct {
	Conn *websocket.Conn
	valid bool
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}
var channels = make([]Channel, 0, 10)

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	connWrapper := &ConnWrapper{conn, true}
	conn.SetCloseHandler(connWrapper.WsCloseHandler)
	for {
		if ! connWrapper.valid {
			break
		}
		_, msgByte, _ := conn.ReadMessage()
		receive := string(msgByte)
		if receive == "" {
			continue
		}
		log.Println("receive", receive)
		if strings.HasPrefix(receive, "open@") {
			channelName := strings.Replace(receive, "open@", "", 1)
			if channelName == "" {
				continue
			}
			channelExists := false
			for index := range channels {
				channel := &channels[index]
				if channelName == channel.Name {
					channelExists = true
					channel.Sockets = append(channel.Sockets, conn)
					_ = conn.WriteMessage(websocket.TextMessage, []byte(channel.Content))
					break
				}
			}
			if ! channelExists {
				welcomeMsg := "Welcome, this is a new channel"
				sockets := make([]*websocket.Conn, 0, 10)
				sockets = append(sockets, conn)
				channels = append(channels, Channel{
					Name: channelName,
					Content: welcomeMsg,
					Sockets: sockets,
				})
				_ = conn.WriteMessage(websocket.TextMessage, []byte(welcomeMsg))
			}
		} else {
			receiveArr := strings.Split(receive, ":")
			channelName := receiveArr[0]
			var content string
			if len(receiveArr) == 1 {
				content = ""
			} else {
				content = strings.Join(receiveArr[1:], ":")
			}
			for index := range channels {
				channel := &channels[index]
				if channelName == channel.Name {
					channel.Content = content
					for _, socket := range channel.Sockets {
						_ = socket.WriteMessage(websocket.TextMessage, []byte(content))
					}
					break
				}
			}
		}
	}
}

func (conn *ConnWrapper) WsCloseHandler(_ int, _ string) error {
	conn.valid = false
	for channelIndex := range channels {
		channel := &channels[channelIndex]
		for sockIndex, socket := range channel.Sockets {
			if conn.Conn == socket {
				log.Println("socket disconnected", channel.Name)
				channel.Sockets = append(channel.Sockets[:sockIndex], channel.Sockets[sockIndex+1:]...)
				break
			}
		}
		if len(channel.Sockets) == 0 {
			channels = append(channels[:channelIndex], channels[channelIndex+1:]...)
		}
	}
	return nil
}

func HttpIndexHandler(w http.ResponseWriter, _ *http.Request) {
	indexHtml, _ := ioutil.ReadFile("index.html")
	_, _ = w.Write(indexHtml)
}

func HttpStatusHandler(w http.ResponseWriter, _ *http.Request) {
	var html string
	for _, channel := range channels {
		html += channel.Name + ", socks count: " + strconv.Itoa(len(channel.Sockets)) + "\n"
	}
	_, _ = w.Write([]byte(html))
}


func main() {
	http.HandleFunc("/ws", WebSocketHandler)
	http.HandleFunc("/status", HttpStatusHandler)
	http.HandleFunc("/", HttpIndexHandler)
	log.Println("start")
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
