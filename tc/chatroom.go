package tc

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

type ChatroomConnectionData struct {
	Result   string
	Endpoint string
}

type ChatroomJoin struct {
	Tc        string `json:"tc"`
	Req       int    `json:"req"`
	Useragent string `json:"useragent"`
	Token     string `json:"token"`
	Room      string `json:"room"`
	Nick      string `json:"nick"`
}

const TcTokenUrl = "/api/v1.0/room/token/"

var done chan interface{}
var interrupt chan os.Signal

func JoinChatroom(username string, password string, nickname string, chatroom string) {
	tcClient := Login(username, password)

	connectionData := loadChatroomConnectionData(&tcClient, chatroom)

	connectToChatroom(username, nickname, chatroom, &connectionData, tcClient.cookies)
}

func loadChatroomConnectionData(client *tcClient, chatroom string) ChatroomConnectionData {
	tokenUrl := TcHost + TcTokenUrl + chatroom

	request, err := http.NewRequest("GET", tokenUrl, nil)
	if err != nil {
		log.Panic("aaaaaaa")
	}

	for _, cookie := range client.cookies {
		request.AddCookie(cookie)
	}

	requestClient := &http.Client{}

	resp, err := requestClient.Do(request)
	if err != nil {
		log.Panic("bitch")
	}
	defer resp.Body.Close()

	rawData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Panic("bitch")
	}
	var data ChatroomConnectionData
	json.Unmarshal(rawData, &data)
	return data
}

func connectToChatroom(username string, nickname string, chatroom string, connectionData *ChatroomConnectionData, cookies []*http.Cookie) {
	done = make(chan interface{})
	interrupt = make(chan os.Signal)

	signal.Notify(interrupt, os.Interrupt)

	log.Print("connecting to: " + connectionData.Endpoint)
	log.Printf("n cookies: %f", len(cookies))

	cookieHeader := buildCookieHeader(cookies)

	log.Print("cookieHeader: " + cookieHeader)

	httpHeaders := http.Header{}
	httpHeaders.Add("Cookie", cookieHeader)

	conn, _, err := websocket.DefaultDialer.Dial(connectionData.Endpoint, httpHeaders)
	if err != nil {
		log.Fatal("Error connecting to Websocket Server:", err)
	}

	defer conn.Close()

	go receiveHandler(conn)

	chatroomJoin := ChatroomJoin{
		Tc:        "join",
		Req:       1,
		Useragent: "tinychat-client-webrtc-undefined_linux x86_64-2.0.20-420",
		Token:     connectionData.Result,
		Room:      chatroom,
		Nick:      nickname,
	}

	sendJoin(conn, chatroomJoin)

	for {
		select {
		case <-time.After(time.Duration(1) * time.Millisecond * 1000):
			// Send an echo packet every second
			err := conn.WriteMessage(websocket.TextMessage, []byte("Hello from GolangDocs!"))
			if err != nil {
				log.Println("Error during writing to websocket:", err)
				return
			}

		case <-interrupt:
			// We received a SIGINT (Ctrl + C). Terminate gracefully...
			log.Println("Received SIGINT interrupt signal. Closing all pending connections")

			// Close our websocket connection
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Error during closing websocket:", err)
				return
			}

			select {
			case <-done:
				log.Println("Receiver Channel Closed! Exiting....")
			case <-time.After(time.Duration(1) * time.Second):
				log.Println("Timeout in closing receiving channel. Exiting....")
			}
			return
		}
	}
}

func receiveHandler(connection *websocket.Conn) {
	defer close(done)
	for {
		_, msg, err := connection.ReadMessage()
		if err != nil {
			log.Println("Error in receive:", err)
			return
		}
		log.Printf("Received: %s\n", msg)
	}
}

func sendJoin(conn *websocket.Conn, joinData ChatroomJoin) {
	//bytes, err := json.Marshal(joinData)
	//log.Print("sending: " + string(bytes))
	//if err != nil {
	//	panic(err)
	//}
	err := conn.WriteJSON(joinData)
	if err != nil {
		panic(err)
	}
	//log.Print("sending request body: " + string(bytes))
}

func buildCookieHeader(cookies []*http.Cookie) string {
	rawCookies := []string{}
	for _, cookie := range cookies {
		if cookie.Name == "hash" || cookie.Name == "pass" || cookie.Name == "user" {
			c := cookie.Name + ":" + cookie.Value
			log.Print("raw cookie: " + c)
			rawCookies = append(rawCookies, c)
		}
	}
	return strings.Join(rawCookies, "; ")
}
