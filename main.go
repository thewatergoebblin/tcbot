package main

import (
	"gobot/conf"
	"gobot/tc"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

const defaultEnv = "default"

func main() {

	cliArgs := parseCommandLineArgs()

	env := selectEnv(cliArgs)

	config := conf.LoadConfiguration(env)

	username, password, nickname := readPasswordFile(config.PasswordFile)

	tc.JoinChatroom(config.Proxy, username, password, nickname, config.DefaultChatroom)
}

func selectEnv(cliArgs map[string]string) string {
	env := cliArgs["env"]
	if env == "" {
		env = os.Getenv("TC_ENV")
		if env == "" {
			return defaultEnv
		}
	}
	return env
}

func parseCommandLineArgs() map[string]string {
	results := make(map[string]string)
	args := os.Args
	if len(args) > 2 {
		log.Panic("No more than 2 command line arguments expected")
	}
	for _, a := range args[1:] {
		if strings.HasPrefix(a, "--") {
			argSplit := strings.Split(a, "=")
			if argSplit[0] == "--env" {
				results["env"] = argSplit[1]
			} else {
				log.Panic("Unknown command line argument: " + argSplit[0])
			}
		} else {
			log.Panic("Command line arguments must begin with --: ", a)
		}
	}
	return results
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
