package conf

import (
	"encoding/json"
	"fmt"
	"gobot/tc"
	"io/ioutil"
	"log"
	"os"
)

type TcConfig struct {
	Proxy           *tc.TcProxy `json:"proxy"`
	PasswordFile    string      `json:"passwordFile"`
	DefaultChatroom string      `json:"defaultChatroom"`
}

const configRoot = "resources/conf"

func LoadConfiguration(env string) TcConfig {
	fileName := fmt.Sprintf("%s/%s.json", configRoot, env)
	configFile, err := os.Open(fileName)
	if err != nil {
		log.Panic("Failed to open configuration file: "+fileName, err)
	}
	fileBytes, err := ioutil.ReadAll(configFile)
	if err != nil {
		log.Panic("Failed to read configuration file: "+fileName, err)
	}
	defer configFile.Close()

	var config TcConfig
	err = json.Unmarshal(fileBytes, &config)
	if err != nil {
		log.Panic("Failed to parse configuration file: "+fileName, err)
	}
	return config
}
