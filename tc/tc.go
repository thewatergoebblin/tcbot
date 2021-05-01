package tc

import (
	"github.com/PuerkitoBio/goquery"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const TcHost = "https://tinychat.com"

type tcClient struct {
	host    string
	cookies map[string]string
}

func Login(username string, password string) tcClient {

	const redirectUrl = TcHost + "/home"

	doc, cookies := loadSignOnData(redirectUrl)

	token := parseLoginToken(doc)

	request := buildLoginRequest(username, password, token, cookies)
	client := &http.Client{}

	resp, err := client.Do(request)
	if err != nil {
		log.Panic("shit")
	}

	defer resp.Body.Close()

	//body, err := io.ReadAll(resp.Body)
	//if err != nil {
	//	log.Panic("error happened")
	//}

	//bodyStr := string(body)
	//log.Print("result: " + bodyStr)

	log.Print("-status: " + resp.Status)

	return tcClient{username, make(map[string]string)}
}

func buildLoginRequest(username string, password string, token string, cookies []*http.Cookie) *http.Request {
	const url = TcHost + "/login"
	formData := makeLoginForm(username, password, token)
	formDataEncoded := formData.Encode()
	request, err := http.NewRequest("POST", url, strings.NewReader(formDataEncoded))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	log.Print("request: " + formDataEncoded)
	if err != nil {
		log.Panic("error happened")
	}
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}

	return request
}

func loadSignOnData(redirect string) (*goquery.Document, []*http.Cookie) {
	url := TcHost + "/start?" + redirect
	resp, err := http.Get(url)

	if err != nil {
		log.Panic("error happened")
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Panic("error happened")
	}
	return doc, resp.Cookies()
}

func parseLoginToken(doc *goquery.Document) string {
	tokenNode := doc.Find("#form-signin > input[name='_token']")
	token, exists := tokenNode.Attr("value")

	if !exists {
		log.Panic("token not found")
	}
	return token
}

func makeLoginForm(username string, password string, token string) url.Values {
	return url.Values{
		"login_username": {username},
		"login_password": {password},
		"remember":       {"1"},
		"next":           {"https://tinychat.com/home"},
		"_token":         {token},
	}
}
