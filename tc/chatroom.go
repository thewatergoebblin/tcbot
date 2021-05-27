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

type Pong struct {
	Tc  string `json:"tc"`
	Req int    `json:"req"`
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
var pong chan interface{}

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

	reqCounter := 1

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
		Req:       reqCounter,
		Useragent: "DoD Missile Silo",
		Token:     connectionData.Result,
		Room:      chatroom,
		Nick:      nickname,
	}

	sendJoin(conn, chatroomJoin)

	reqCounter++

	for {
		select {
		case <-pong:
			sendPong(conn, reqCounter)
			reqCounter++
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
	for {
		var payload map[string]interface{}
		err := connection.ReadJSON(&payload)
		if err != nil {
			log.Panic("Error in receive:", err)
		}
		b, _ := json.Marshal(payload)
		log.Println(string(b))
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
		readUserData(payload)
	case "joined":
		return
	case "ping":
		pong <- "pong"
	default:
		log.Print("warning - unhandled tc message type: ", tc)
	}
}

func readUserData(payload map[string]interface{}) User {
	usersJson := payload["users"].([]interface{})
	for _, userInt := range usersJson {
		user := userInt.(map[string]interface{})
		u := User{
			AchievementUrl: user["achievement_url"].(string),
		}
		log.Println("user: ", u.AchievementUrl)
	}
	return User{}
}

func sendPong(conn *websocket.Conn, req int) {
	log.Printf("sending pong %d\n", req)
	pongReq := Pong{
		Tc:  "pong",
		Req: req,
	}
	err := conn.WriteJSON(pongReq)
	if err != nil {
		log.Panic("Failed to send pong message: ", err)
	}
}
