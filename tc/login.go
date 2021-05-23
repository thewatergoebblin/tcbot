package tc

import (
	"github.com/PuerkitoBio/goquery"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const TcHost = "https://tinychat.com"

type TcClient struct {
	cookies []*http.Cookie
}

func Login(username string, password string) TcClient {
	return LoginAndRedirect(username, password, "")
}

func LoginAndRedirect(username string, password string, redirect string) TcClient {
	doc, cookies := loadSignOnData(redirect)

	token := parseLoginToken(doc)
	request := buildLoginRequest(username, password, redirect, token, cookies)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(request)
	if err != nil {
		log.Panic("Failed to login to tinychat - request failed: ", err)
	}

	defer resp.Body.Close()

	return TcClient{resp.Cookies()}
}

func buildLoginRequest(username string, password string, redirect, token string, cookies []*http.Cookie) *http.Request {
	const url = TcHost + "/login"
	formData := makeLoginForm(username, password, redirect, token)
	formDataEncoded := formData.Encode()

	request, err := http.NewRequest("POST", url, strings.NewReader(formDataEncoded))
	if err != nil {
		log.Panic("Failed to login to tinychat - failed to build login request: ", err)
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	return request
}

func loadSignOnData(redirect string) (*goquery.Document, []*http.Cookie) {
	url := TcHost + "/start?" + redirect
	resp, err := http.Get(url)

	if err != nil {
		log.Panic("Failed to load initial tinychat page - request failed", err)
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Panic("Failed to parse initial tinychat page: ", err)
	}
	return doc, resp.Cookies()
}

func parseLoginToken(doc *goquery.Document) string {
	tokenNode := doc.Find("#form-signin > input[name='_token']")
	token, exists := tokenNode.Attr("value")
	if !exists {
		log.Panic("Failed to acquire necessary token for login")
	}
	return token
}

func makeLoginForm(username string, password string, redirect string, token string) url.Values {
	return url.Values{
		"login_username": {username},
		"login_password": {password},
		"remember":       {"1"},
		"next":           {redirect},
		"_token":         {token},
	}
}
