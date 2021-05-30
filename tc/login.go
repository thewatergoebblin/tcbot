package tc

import (
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/proxy"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const TcHost = "https://tinychat.com"

type TcClient struct {
	cookies []*http.Cookie
	client  *http.Client
	tcProxy *TcProxy
}

type TcProxy struct {
	Host     string
	Username string
	Password string
}

func Login(tcProxy *TcProxy, username string, password string) TcClient {
	return LoginAndRedirect(tcProxy, username, password, "")
}

func LoginProxy(tcProxy *TcProxy, username string, password string) TcClient {
	return LoginAndRedirect(tcProxy, username, password, "")
}

func LoginAndRedirect(tcProxy *TcProxy, username string, password string, redirect string) TcClient {
	client := buildHttpClient(tcProxy)
	doc, cookies := loadSignOnData(client, redirect)
	token := parseLoginToken(doc)
	request := buildLoginRequest(username, password, redirect, token, cookies)
	resp, err := client.Do(request)

	if err != nil {
		log.Panic("Failed to login to tinychat - request failed: ", err)
	}

	defer resp.Body.Close()

	return TcClient{
		cookies: resp.Cookies(),
		client:  client,
		tcProxy: tcProxy,
	}
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

func loadSignOnData(client *http.Client, redirect string) (*goquery.Document, []*http.Cookie) {
	url := TcHost + "/start?" + redirect
	resp, err := client.Get(url)

	if err != nil {
		log.Panic("Failed to load initial tinychat page - request failed: ", err)
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

func buildHttpClient(tcProxy *TcProxy) *http.Client {
	if tcProxy != nil {
		dialSocksProxy, err := proxy.SOCKS5("tcp", tcProxy.Host, nil, nil)
		if err != nil {
			log.Panic("Failed to create proxy object")
		}
		transport := http.Transport{
			Dial: dialSocksProxy.Dial,
		}
		return &http.Client{
			Transport: &transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	} else {
		return &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
}
