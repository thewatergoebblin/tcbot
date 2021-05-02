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

type chatroomConnectionData struct {
	Result   string
	Endpoint string
}

const TcTokenUrl = "/api/v1.0/room/token/"

var done chan interface{}
var interrupt chan os.Signal

func JoinChatroom(username string, password string, chatroom string) {
	tcClient := Login(username, password)

	connectionData := loadChatroomConnectionData(chatroom)

	connectToChatroom(connectionData.Endpoint, tcClient.cookies)
}

func loadChatroomConnectionData(chatroom string) chatroomConnectionData {
	tokenUrl := TcHost + TcTokenUrl + chatroom
	resp, err := http.Get(tokenUrl)
	if err != nil {
		log.Panic("bitch")
	}
	defer resp.Body.Close()

	rawData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Panic("bitch")
	}
	var data chatroomConnectionData
	json.Unmarshal(rawData, &data)
	return data
}

func connectToChatroom(url string, cookies []*http.Cookie) {
	done = make(chan interface{})
	interrupt = make(chan os.Signal)

	signal.Notify(interrupt, os.Interrupt) // Notify the interrupt channel for SIGINT

	log.Print("connecting to: " + url)
	log.Printf("n cookies: %f", len(cookies))

	cookieHeader := buildCookieHeader(cookies)

	httpHeaders := http.Header{}
	httpHeaders.Add("Cookie", cookieHeader)

	conn, _, err := websocket.DefaultDialer.Dial(url, httpHeaders)
	if err != nil {
		log.Fatal("Error connecting to Websocket Server:", err)
	}
	defer conn.Close()
	go receiveHandler(conn)

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

func buildCookieHeader(cookies []*http.Cookie) string {
	rawCookies := []string{}
	for _, cookie := range cookies {
		rawCookies = append(rawCookies, cookie.Raw)
	}
	return strings.Join(rawCookies, ";")
}
