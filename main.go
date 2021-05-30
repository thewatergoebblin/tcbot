package main

import (
	"gobot/tc"
	"io/ioutil"
	"log"
	"strings"
)

func main() {
	const passwordFile = "password.pwd"
	username, password, nickname := readPasswordFile(passwordFile)

	tcProxy := tc.TcProxy{
		Host: "localhost:9050",
	}

	tc.JoinChatroom(&tcProxy, username, password, nickname, "littlebunny")
}

func readPasswordFile(fileName string) (username string, password string, nickname string) {
	result, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Panic("Failed to read password file: ", err)
	}
	resultStr := string(result)
	credentials := strings.Fields(resultStr)
	return credentials[0], credentials[1], credentials[2]
}
