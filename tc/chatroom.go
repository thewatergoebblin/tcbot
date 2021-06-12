package tc

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
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
	GiftPoints     int    `json:"giftpoints"`
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
	userNickMap   map[string]*User
	userHandleMap map[int]*User
	userCamMap    map[int]*User
}

const TcTokenUrl = "/api/v1.0/room/token/"

var done chan interface{}
var interrupt chan os.Signal
var pong chan interface{}

func JoinChatroom(tcProxy *TcProxy, username string, password string, nickname string, chatroom string) {

	log.Print("----- Logging In -----")

	tcClient := Login(tcProxy, username, password)

	log.Print("----- Loading Chatroom Connection Data -----")

	connectionData := loadChatroomConnectionData(&tcClient, chatroom)

	log.Print("------ Joining to Chatroom -----")

	connectToChatroom(&tcClient, username, nickname, chatroom, &connectionData)
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

	resp, err := tcClient.client.Do(request)
	if err != nil {
		log.Panic("Failed to load chatroom connection data - request failed: ", err)
	}
	defer resp.Body.Close()

	rawData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Panic("Failed to load chatroom connection data - ", err)
	}
	var data ChatroomConnectionData
	err = json.Unmarshal(rawData, &data)
	if err != nil {
		log.Panic("Failed to load chatroom connection data - parsing json failed - ", err)
	}
	return data
}

func connectToChatroom(tcClient *TcClient, username string, nickname string, chatroom string, connectionData *ChatroomConnectionData) {
	done = make(chan interface{})
	interrupt = make(chan os.Signal)
	pong = make(chan interface{})

	reqCounter := 1

	signal.Notify(interrupt, os.Interrupt)

	var dialer websocket.Dialer

	if tcClient.tcProxy == nil {
		dialer = websocket.Dialer{}
	} else {
		log.Print("Connecting over socks5 proxy")
		proxyDialer, err := proxy.SOCKS5("tcp", "localhost:9050", nil, nil)
		if err != nil {
			log.Panic("Failed to construct proxy dialer", err)
		}
		dialer = websocket.Dialer{NetDial: proxyDialer.Dial}
	}

	conn, _, err := dialer.Dial(connectionData.Endpoint, nil)
	if err != nil {
		log.Fatal("Error connecting to chatroom - websocket connection failed: ", err)
	}

	defer conn.Close()

	chatroomState := ChatroomState{
		userNameMap:   make(map[string]*User),
		userNickMap:   make(map[string]*User),
		userHandleMap: make(map[int]*User),
		userCamMap:    make(map[int]*User),
	}

	go chatroomState.receiveHandler(conn)

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

func (state ChatroomState) receiveHandler(connection *websocket.Conn) {
	defer close(done)
	for {
		var payload map[string]interface{}
		err := connection.ReadJSON(&payload)
		if err != nil {
			log.Panic("Error in receive:", err)
		}
		raw, err := json.Marshal(payload)
		rawStr := string(raw)
		if err != nil {
			log.Println("Error marshalling json data: ", string(raw),
				" with error: ", err)
		}
		state.readInboundMessage(payload, rawStr)
	}
}

func sendJoin(conn *websocket.Conn, joinData ChatroomJoin) {
	err := conn.WriteJSON(joinData)
	if err != nil {
		log.Panic("Failed to send join message: ", err)
	}
}

func (state ChatroomState) readInboundMessage(payload map[string]interface{},
	raw string) {
	tc := payload["tc"].(string)
	switch tc {
	case "userlist":
		state.readUserData(payload)
		return
	case "join":
		state.handleJoin(payload)
		return
	case "quit":
		state.handleQuit(payload)
		return
	case "publish":
		state.handlePublish(payload)
		return
	case "unpublish":
		state.handleUnpublish(payload)
		return
	case "ping":
		pong <- "pong"
	case "msg":
		state.handleMessage(payload)
		return
	case "joined":
		return
	default:
		log.Print("warning - unhandled tc message type: ", tc, " raw: ", raw)
	}
}

func (state ChatroomState) readUserData(payload map[string]interface{}) {
	usersJson := payload["users"].([]interface{})
	for _, userInt := range usersJson {
		userMap := userInt.(map[string]interface{})
		user := parseUser(userMap)
		state.addUser(user)
	}
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

func parseUser(userMap map[string]interface{}) *User {
	return &User{
		AchievementUrl: userMap["achievement_url"].(string),
		Avatar:         userMap["avatar"].(string),
		Featured:       userMap["featured"].(bool),
		GiftPoints:     int(userMap["giftpoints"].(float64)),
		Handle:         int(userMap["handle"].(float64)),
		Lurker:         userMap["lurker"].(bool),
		Mod:            userMap["mod"].(bool),
		Nick:           userMap["nick"].(string),
		Owner:          userMap["owner"].(bool),
		SessionId:      userMap["session_id"].(string),
		Subscription:   int(userMap["subscription"].(float64)),
		Username:       userMap["username"].(string),
	}
}

func (state ChatroomState) handleJoin(payload map[string]interface{}) {
	user := parseUser(payload)
	state.addUser(user)
}

func (state ChatroomState) handleQuit(payload map[string]interface{}) {
	handle := int(payload["handle"].(float64))
	state.removeUser(handle)
}

func (state ChatroomState) handlePublish(payload map[string]interface{}) {
	handle := int(payload["handle"].(float64))
	user := state.userHandleMap[handle]
	state.userCamMap[handle] = user
	log.Printf("user %s:%s cammed up\n", user.Username, user.Nick)
}

func (state ChatroomState) handleUnpublish(payload map[string]interface{}) {
	handle := int(payload["handle"].(float64))
	if user, exists := state.userHandleMap[handle]; exists {
		delete(state.userCamMap, handle)
		log.Printf("user %s:%s cammed down\n", user.Username, user.Nick)
	}
}

func (state ChatroomState) handleMessage(payload map[string]interface{}) {
	handle := int(payload["handle"].(float64))
	text := payload["text"].(string)
	user := state.userHandleMap[handle]
	log.Printf("message from: %s:%s :: %s\n", user.Username, user.Nick, text)
}

func (state ChatroomState) addUser(user *User) {
	state.userNameMap[user.Username] = user
	state.userNickMap[user.Nick] = user
	state.userHandleMap[user.Handle] = user
	log.Printf("user joined: %s:%s\n", user.Username, user.Nick)
}

func (state ChatroomState) removeUser(handle int) {
	user := state.userHandleMap[handle]
	delete(state.userHandleMap, handle)
	delete(state.userNickMap, user.Nick)
	delete(state.userNameMap, user.Username)
	if _, exists := state.userCamMap[handle]; exists {
		delete(state.userCamMap, handle)
	}
	log.Printf("user quit: %s:%s\n", user.Username, user.Nick)
}

func (state ChatroomState) handleJoined(payload map[string]interface{}) {
	//TODO: implement
}
