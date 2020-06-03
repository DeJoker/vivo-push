package vivopush

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var authToken *AuthToken = new(AuthToken)

type VivoClient struct {
	AppId     string
	AppKey    string
	AppSecret string
}

type VivoTokenPar struct {
	AppId     string `json:"appId"`
	AppKey    string `json:"appKey"`
	Timestamp int64  `json:"timestamp"`
	Sign      string `json:"sign"`
}

type AuthToken struct {
	token      string
	valid_time int64
}

type VivoPush struct {
	host       string
	Auth_token string
}


var (
	client = &http.Client{
		Timeout : time.Second * 60,
	}
)


func NewClient(appId, appKey, appSecret string) (*VivoPush, error) {
	vc := &VivoClient{
		appId,
		appKey,
		appSecret,
	}
	token, err := vc.GetToken()
	if err != nil {
		return nil, err
	}
	return &VivoPush{
		host:       ProductionHost,
		Auth_token: token,
	}, nil
}

//----------------------------------------Token----------------------------------------//
//获取token  返回的expiretime 秒  当过期的时候
func (vc *VivoClient) GetToken() (string, error) {
	now := time.Now().UnixNano() / 1e6
	if authToken != nil {
		if authToken.valid_time > now {
			return authToken.token, nil
		}
	}
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(vc.AppId + vc.AppKey + strconv.FormatInt(now, 10) + vc.AppSecret))
	sign := hex.EncodeToString(md5Ctx.Sum(nil))

	formData, err := json.Marshal(&VivoTokenPar{
		AppId:     vc.AppId,
		AppKey:    vc.AppKey,
		Timestamp: now,
		Sign:      sign,
	})
	if err != nil {
		return "", err
	}


	req, err := http.NewRequest("POST", ProductionHost+AuthURL, bytes.NewReader(formData))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	res, err := handleResponse(resp)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("HTTP status code:"+strconv.Itoa(resp.StatusCode))
	}

	var result TokenResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return "", err
	}
	if result.Result != 0 {
		return "", errors.New(fmt.Sprintf("%s", res))
	}

	authToken.token = result.AuthToken
	authToken.valid_time = now + 3600000 //1小时有效
	return result.AuthToken, nil
}

//----------------------------------------Sender----------------------------------------//
// 根据regID，发送消息到指定设备上
func (v *VivoPush) Send(msg *Message, regID string) (*SendResult, error) {
	params := v.assembleSendParams(msg, regID)
	res, err := v.doPost(v.host+SendURL, params)
	if err != nil {
		return nil, err
	}
	var result SendResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}
	if result.Result != 0 {
		return &result, errors.New(fmt.Sprintf("%s", res))
	}
	return &result, nil
}

// 保存群推消息公共体接口
func (v *VivoPush) SaveListPayload(msg *MessagePayload) (*SendResult, error) {
	res, err := v.doPost(v.host+SaveListPayloadURL, msg.JSON())
	if err != nil {
		return nil, err
	}
	var result SendResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}
	if result.Result != 0 {
		return &result, errors.New(fmt.Sprintf("%s", res))
	}
	return &result, nil
}

// 群推
func (v *VivoPush) SendList(msg *MessagePayload, regIds []string) (*SendResult, error) {
	if len(regIds) < 2 || len(regIds) > 1000 {
		return nil, errors.New("regIds个数必须大于等于2,小于等于 1000")
	}
	saveResult, err := v.SaveListPayload(msg)
	if err != nil {
		return nil, err
	}
	// save中已经判断过saveResult的code了
	bytes, err := json.Marshal(NewListMessage(regIds, saveResult.TaskId))
	if err != nil {
		return nil, err
	}

	//推送
	pushRes, err := v.doPost(v.host+PushToListURL, bytes)
	if err != nil {
		return nil, err
	}
	var result SendResult
	err = json.Unmarshal(pushRes, &result)
	if err != nil {
		return nil, err
	}
	if result.Result != 0 {
		return &result, errors.New(fmt.Sprintf("%s", pushRes))
	}
	return &result, nil
}

// 全量推送
func (v *VivoPush) SendAll(msg *MessagePayload) (*SendResult, error) {
	res, err := v.doPost(v.host+PushToAllURL, msg.JSON())
	if err != nil {
		return nil, err
	}
	var result SendResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}
	if result.Result != 0 {
		return &result, errors.New(fmt.Sprintf("%s", res))
	}
	return &result, nil
}

//----------------------------------------Tracer----------------------------------------//
// 获取指定消息的状态。
func (v *VivoPush) GetMessageStatusByJobKey(jobKey string) (*BatchStatusResult, error) {
	params := v.assembleStatusByJobKeyParams(jobKey)
	res, err := v.doGet(v.host+MessagesStatusURL, params)
	if err != nil {
		return nil, err
	}
	var result BatchStatusResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return &result, err
	}
	return &result, nil
}

func (v *VivoPush) assembleSendParams(msg *Message, regID string) []byte {
	msg.RegId = regID
	jsondata := msg.JSON()
	return jsondata
}

func (v *VivoPush) assembleStatusByJobKeyParams(jobKey string) string {
	form := url.Values{}
	form.Add("taskIds", jobKey)
	return "?" + form.Encode()
}

func handleResponse(response *http.Response) ([]byte, error) {
	defer func() {
		response.Body.Close()
	}()

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (v *VivoPush) doPost(url string, formData []byte) ([]byte, error) {
	var result []byte
	var req *http.Request
	var resp *http.Response
	var err error

	req, err = http.NewRequest("POST", url, bytes.NewReader(formData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authToken", v.Auth_token)

	tryTime := 0
tryAgain:
	resp, err = client.Do(req)
	if err != nil {
		tryTime += 1
		if tryTime < PostRetryTimes {
			goto tryAgain
		}
		return nil, err
	}
	result, err = handleResponse(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("HTTP status code:"+strconv.Itoa(resp.StatusCode))
	}
	return result, nil
}

func (v *VivoPush) doGet(url string, params string) ([]byte, error) {
	var result []byte
	var req *http.Request
	var resp *http.Response
	var err error
	req, err = http.NewRequest("GET", url+params, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authToken", v.Auth_token)

	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	result, err = handleResponse(resp)
	if err != nil {
		return nil, err
	}
	return result, nil
}
