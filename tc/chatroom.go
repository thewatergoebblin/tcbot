package tc

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
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

type User struct {
	AchievementUrl string `json:"achievement_url"`
	Avatar         string `json:"avatar"`
	Featured       bool   `json:"featured"`
	giftPoints     string `json:"giftpoints"`
	Handle         int    `json:"handle"`
	Lurker         bool   `json:"lurker"`
	Mod            bool   `json:"mod"`
	Nick           string `json:"nick"`
	Owner          bool   `json:"owner"`
	SessionId      string `json:"session_id"`
	Subscription   int    `json:"subscription"`
	Username       string `json:"username"`
}

type ChatroomState struct {
	userNameMap   map[string]*User
	userHandleMap map[int]*User
}

const TcTokenUrl = "/api/v1.0/room/token/"

var done chan interface{}
var interrupt chan os.Signal

func JoinChatroom(username string, password string, nickname string, chatroom string) {

	log.Print("----- Logging In -----")

	tcClient := Login(username, password)

	log.Print("----- Loading Chatroom Connection Data -----")

	connectionData := loadChatroomConnectionData(&tcClient, chatroom)

	log.Print("------ Joining to Chatroom -----")

	connectToChatroom(username, nickname, chatroom, &connectionData)
}

func loadChatroomConnectionData(tcClient *TcClient, chatroom string) ChatroomConnectionData {
	tokenUrl := TcHost + TcTokenUrl + chatroom

	request, err := http.NewRequest("GET", tokenUrl, nil)
	if err != nil {
		log.Panic("Failed to load chatroom connection data - constructing request object failed: ", err)
	}

	for _, cookie := range tcClient.cookies {
		request.AddCookie(cookie)
	}

	requestClient := &http.Client{}

	resp, err := requestClient.Do(request)
	if err != nil {
		log.Panic("Failed to load chatroom connection data - request failed: ", err)
	}
	defer resp.Body.Close()

	rawData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Panic("Failed to load chatroom connection data - ", err)
	}
	var data ChatroomConnectionData
	json.Unmarshal(rawData, &data)
	return data
}

func connectToChatroom(username string, nickname string, chatroom string, connectionData *ChatroomConnectionData) {
	done = make(chan interface{})
	interrupt = make(chan os.Signal)

	signal.Notify(interrupt, os.Interrupt)

	conn, _, err := websocket.DefaultDialer.Dial(connectionData.Endpoint, nil)
	if err != nil {
		log.Fatal("Error connecting to chatroom - websocket connection failed: ", err)
	}

	defer conn.Close()

	chatroomState := ChatroomState{
		userNameMap:   make(map[string]*User),
		userHandleMap: make(map[int]*User),
	}

	go receiveHandler(&chatroomState, conn)

	chatroomJoin := ChatroomJoin{
		Tc:        "join",
		Req:       1,
		Useragent: "DoD Missile Silo",
		Token:     connectionData.Result,
		Room:      chatroom,
		Nick:      nickname,
	}

	sendJoin(conn, chatroomJoin)

	for {
		select {
		case <-interrupt:
			log.Println("Received SIGINT interrupt signal. Closing all pending connections")
			err := conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Panic("Error during closing websocket:", err)
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

func receiveHandler(state *ChatroomState, connection *websocket.Conn) {
	defer close(done)
	var payload map[string]interface{}
	for {
		err := connection.ReadJSON(&payload)
		if err != nil {
			log.Panic("Error in receive:", err)
		}
		readInboundMessage(state, payload)
	}
}

func sendJoin(conn *websocket.Conn, joinData ChatroomJoin) {
	err := conn.WriteJSON(joinData)
	if err != nil {
		log.Panic("Failed to send join message: ", err)
	}
}

func readInboundMessage(state *ChatroomState, payload map[string]interface{}) {
	tc := payload["tc"].(string)
	switch tc {
	case "userlist":
		log.Print("got userlist")
	default:
		log.Print("warning - unhandled tc message type: ", tc)
	}
}
