package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"gopkg.in/gcfg.v1"
)

var accessToken string
var agentId int

var (
	configFile  = flag.String("config", "weixin.ini", "General weixin configuration file")
	users       = flag.String("users", "AndrewDi", "Send targets")
	message     = flag.String("msg", "You haven't set main message body", "Message body")
	profileName = flag.String("profile", "Dev", "Weixin App config name")
	nocache     = flag.Bool("nocache", false, "If cache AccessToken to file.")
)

func main() {
	flag.Parse()

	config := struct {
		Profile map[string]*struct {
			Corpid     string
			Corpsecret string
			AgentId    int
		}
	}{}

	err := gcfg.ReadFileInto(&config, *configFile)
	if err != nil {
		panic(err)
	}

	agentId = config.Profile[*profileName].AgentId
	err = getAccessToken(config.Profile[*profileName].Corpid, config.Profile[*profileName].Corpsecret)
	if err != nil {
		panic(err)
	}
	ret, err := sendTextMsg(*message, *users)
	if err != nil {
		fmt.Printf("Send message fail:%s Error:%s", ret, err.Error())
	}
	fmt.Print(ret)
}

type ReturnMsg struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

type AccessTokenResponse struct {
	ReturnMsg
	AccessToken string    `json:"access_token"`
	ExpiresIn   int       `json:"expires_in"`
	ExpireTime  time.Time `json:"expire_time"`
}

type SendMsgResponse struct {
	ReturnMsg
	InvalidUser string `json:"invaliduser"`
}

type TextMsg struct {
	Content string `json:"content"`
}

type SendMsgRequest struct {
	ToUser  string  `json:"touser"`
	MsgType string  `json:"msgtype"`
	AgentId int     `json:"agentid"`
	Text    TextMsg `json:"text"`
	Safe    int     `json:"safe"`
}

func readTokenCacheFile(filename string) (err error) {
	tokenCache, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	var data AccessTokenResponse
	err = json.Unmarshal(tokenCache, &data)
	if err != nil {
		return
	}
	if time.Now().Before(data.ExpireTime) {
		accessToken = data.AccessToken
	}
	return
}

func getAccessToken(corpid string, corpsecret string) (err error) {
	tokenCacheFile := *profileName + ".tokenCacheFile"
	if corpid == "" || corpsecret == "" {
		panic(fmt.Sprintf("Panic process config corpid:%s corpsecret:%s", corpid, corpsecret))
		return
	}

	err = readTokenCacheFile(tokenCacheFile)
	if err == nil && accessToken != "" {
		return
	}

	client := &http.Client{}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s", corpid, corpsecret)
	request, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return
	}
	response, err := client.Do(request)
	if err != nil || response.StatusCode != 200 {
		return
	}

	defer response.Body.Close()
	var data AccessTokenResponse
	accessTokenBytes, err := ioutil.ReadAll(response.Body)
	err = json.Unmarshal(accessTokenBytes, &data)
	if err != nil {
		return
	}
	if data.ErrCode == 0 {
		accessToken = data.AccessToken

		if *nocache {
			return
		}
		expireDuration, err := time.ParseDuration(fmt.Sprintf("%ds", data.ExpiresIn))
		if err != nil {
			return err
		}
		data.ExpireTime = time.Now().Add(expireDuration)
		tokenCache, err := json.Marshal(data)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(tokenCacheFile, tokenCache, 0700)
	}
	return
}

func sendTextMsg(msg string, users string) (ret string, err error) {
	msgBody := &SendMsgRequest{
		ToUser:  users,
		Text:    TextMsg{Content: msg},
		MsgType: "text",
		AgentId: agentId,
		Safe:    0,
	}

	msgBodyJson, err := json.Marshal(msgBody)
	client := &http.Client{}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", accessToken)
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(msgBodyJson))
	if err != nil {
		return
	}

	response, err := client.Do(request)
	if err != nil || response.StatusCode != 200 {
		return
	}

	defer response.Body.Close()
	var data SendMsgResponse
	responseBytes, err := ioutil.ReadAll(response.Body)
	err = json.Unmarshal(responseBytes, &data)
	if err != nil {
		ret = response.Status
		return
	}
	ret = data.ErrMsg
	return
}
