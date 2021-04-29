package tc

import "net/http"
import "net/url"
import "log"
import "io"

const TcHost = "https://tinychat.com"

type tcClient struct {
	host    string
	cookies map[string]string
}

func Login(username string, password string) tcClient {
	var payload = makeLoginForm(username, password)
	const url = TcHost + "/login"
	resp, err := http.PostForm(url, payload)

	if err != nil {
		log.Panic("error happened")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Panic("error happened")
	}

	bodyStr := string(body)
	log.Print("result: " + bodyStr)

	log.Print("status: " + resp.Status)

	return tcClient{username, make(map[string]string)}
}

func makeLoginForm(username string, password string) url.Values {
	return url.Values{
		"login_username": {username},
		"login_password": {password},
		"remember":       {"1"},
	}
}
