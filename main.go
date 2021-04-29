package main

import (
	"fmt"
	"gobot/tc"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

func main() {
	const passwordFile = "password.pwd"
	username, password := readPasswordFile(passwordFile)
	tc.Login(username, password)
}

func tcConnect() {
	resp, err := http.Get("https://tinychat.com/api/v1.0/room/token/real9k")
	if err != nil {
		fmt.Println("error")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error")
	}
	bodyStr := string(body)
	fmt.Println(bodyStr)
}

func readPasswordFile(fileName string) (username string, password string) {
	result, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Println("shit")
	}
	resultStr := string(result)
	credentials := strings.Fields(resultStr)
	return credentials[0], credentials[1]
}
