package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/abadojack/whatlanggo"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
)

var goBridge *GoBridge

type GoBridge struct {
	AppName       string
	Port          string
	EmacsPort     string
	Server        *websocket.Conn
	Client        *websocket.Conn
	MessageHandle func(message string)
}

type EmacsMsg struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type Lang struct {
	SourceLangUserSelected string `json:"source_lang_user_selected"`
	TargetLang             string `json:"target_lang"`
}

type CommonJobParams struct {
	WasSpoken    bool   `json:"wasSpoken"`
	TranscribeAS string `json:"transcribe_as"`
	// RegionalVariant string `json:"regionalVariant"`
}

type Params struct {
	Texts           []Text          `json:"texts"`
	Splitting       string          `json:"splitting"`
	Lang            Lang            `json:"lang"`
	Timestamp       int64           `json:"timestamp"`
	CommonJobParams CommonJobParams `json:"commonJobParams"`
}

type Text struct {
	Text                string `json:"text"`
	RequestAlternatives int    `json:"requestAlternatives"`
}

type PostData struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	ID      int64  `json:"id"`
	Params  Params `json:"params"`
}

type ResData struct {
	TransText  string `json:"text"`
	SourceLang string `json:"source_lang"`
	TargetLang string `json:"target_lang"`
}

func main() {
	args := os.Args[1:]
	fmt.Println(args)
	goBridge = NewGoBridge(args[1], args[2], args[3], handleMessage)
	runtime.Goexit()
}

func initData(sourceLang string, targetLang string) *PostData {
	return &PostData{
		Jsonrpc: "2.0",
		Method:  "LMT_handle_texts",
		Params: Params{
			Splitting: "newlines",
			Lang: Lang{
				SourceLangUserSelected: sourceLang,
				TargetLang:             targetLang,
			},
			CommonJobParams: CommonJobParams{
				WasSpoken:    false,
				TranscribeAS: "",
				// RegionalVariant: "en-US",
			},
		},
	}
}

func getICount(translateText string) int64 {
	return int64(strings.Count(translateText, "i"))
}

func getRandomNumber() int64 {
	rand.Seed(time.Now().Unix())
	num := rand.Int63n(99999) + 8300000
	return num * 1000
}

func getTimeStamp(iCount int64) int64 {
	ts := time.Now()
	ms := ts.Sub(time.Unix(0, 0)).Milliseconds()
	if iCount != 0 {
		iCount = iCount + 1
		return ms - ms%iCount + iCount
	} else {
		return ms
	}
}

func translate(content string) (string, error) {
	// create a random id
	id := getRandomNumber()

	reqj := ResData{TransText: content}
	sourceLang := reqj.SourceLang
	targetLang := reqj.TargetLang
	translateText := reqj.TransText
	if sourceLang == "" {
		lang := whatlanggo.DetectLang(translateText)
		deepLLang := strings.ToUpper(lang.Iso6391())
		sourceLang = deepLLang
	}
	if targetLang == "" {
		targetLang = "EN"
	}

	url := "https://www2.deepl.com/jsonrpc"
	id = id + 1
	postData := initData(sourceLang, targetLang)
	text := Text{
		Text:                translateText,
		RequestAlternatives: 3,
	}
	// set id
	postData.ID = id
	// set text
	postData.Params.Texts = append(postData.Params.Texts, text)
	// set timestamp
	postData.Params.Timestamp = getTimeStamp(getICount(translateText))
	post_byte, _ := json.Marshal(postData)
	postStr := string(post_byte)

	// add space if necessary
	if (id+5)%29 == 0 || (id+3)%13 == 0 {
		postStr = strings.Replace(postStr, "\"method\":\"", "\"method\" : \"", -1)
	} else {
		postStr = strings.Replace(postStr, "\"method\":\"", "\"method\": \"", -1)
	}

	post_byte = []byte(postStr)
	reader := bytes.NewReader(post_byte)
	request, err := http.NewRequest("POST", url, reader)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	// Set Headers
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "*/*")
	request.Header.Set("x-app-os-name", "iOS")
	request.Header.Set("x-app-os-version", "16.3.0")
	request.Header.Set("Accept-Language", "en-US,en;q=0.9")
	request.Header.Set("Accept-Encoding", "gzip, deflate, br")
	request.Header.Set("x-app-device", "iPhone13,2")
	request.Header.Set("User-Agent", "DeepL-iOS/2.6.0 iOS 16.3.0 (iPhone13,2)")
	request.Header.Set("x-app-build", "353933")
	request.Header.Set("x-app-version", "2.6")
	request.Header.Set("Connection", "keep-alive")

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	res := gjson.ParseBytes(body)

	if res.Get("error.code").String() == "-32600" {
		fmt.Println(res.Get("error").String())
		return "", fmt.Errorf(res.Get("error").String())
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		fmt.Println("Too Many Requests")
		return "", fmt.Errorf("Too Many Requests")
	}
	return res.Get("result.texts.0.text").String(), nil
}

func handleMessage(message string) {
	fmt.Println("Receviced message:", message)
	var dataArray []interface{}
	if err := json.Unmarshal([]byte(message), &dataArray); err != nil {
		return
	}
	content := dataArray[1].([]interface{})[0].(string)
	style := dataArray[1].([]interface{})[1].(string)
	buffername := dataArray[1].([]interface{})[2].(string)
	placeholder := dataArray[1].([]interface{})[3].(string)
	goBridge.MessageToEmacs("Begin to transalte: " + content)
	translation, err := translate(content)
	emacsCmd := "(insert-translated-name-update-translation-in-buffer \"" + content + "\" \"" + style + "\" \"" + translation + "\" \"" + buffername + "\" \"" + placeholder + "\")"

	if err != nil {
		fmt.Println(err)
		goBridge.MessageToEmacs(err.Error())
		return
	}
	fmt.Println(emacsCmd)
	goBridge.EvalInEmacs(emacsCmd)
}

func NewGoBridge(appName string, port string, emacsPort string, messageHandler func(message string)) *GoBridge {
	d := &GoBridge{
		AppName:       appName,
		Port:          port,
		EmacsPort:     emacsPort,
		MessageHandle: messageHandler,
	}
	d.init()
	return d
}

func (d *GoBridge) init() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println("Failed to upgrade:", err)
			return
		}
		d.Server = conn

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("Failed to read message:", err)
				break
			}

			d.MessageHandle(string(msg[:]))
		}
	})

	go func() {
		http.ListenAndServe(fmt.Sprintf(":%s", d.Port), nil)
	}()

	client, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://127.0.0.1:%s", d.EmacsPort), nil)
	if err != nil {
		fmt.Println("Failed to dial:", err)
		return
	}
	d.Client = client

	fmt.Println("Go bridge connected!")
}

func (d *GoBridge) MessageToEmacs(message string) {
	msg := EmacsMsg{
		Type:    "show-message",
		Content: message,
	}
	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return
	}
	d.Client.WriteMessage(websocket.TextMessage, jsonMsg)
}

func (d *GoBridge) EvalInEmacs(code string) {
	msg := EmacsMsg{
		Type:    "eval-code",
		Content: code,
	}
	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return
	}
	d.Client.WriteMessage(websocket.TextMessage, jsonMsg)
}
