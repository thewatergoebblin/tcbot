package main

import (
	"fmt"
	"gobot/tc"
	"io/ioutil"
	"strings"
)

func main() {
	const passwordFile = "password.pwd"
	username, password := readPasswordFile(passwordFile)
	tc.JoinChatroom(username, password, "real9k")
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
