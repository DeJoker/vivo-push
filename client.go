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

type AuthInfo struct {
	token      string
	validTime  int64
}

type VivoClient struct {
	AppId     string
	AppKey    string
	AppSecret string
	AuthToken AuthInfo
}

type AuthTokenReq struct {
	AppId     string `json:"appId"`
	AppKey    string `json:"appKey"`
	Timestamp int64  `json:"timestamp"`
	Sign      string `json:"sign"`
}



var (
	client = &http.Client{
		Timeout : time.Second * 60,
	}
)


func NewClient(appId, appKey, appSecret string) (*VivoClient, error) {
	vc := &VivoClient{
		appId,
		appKey,
		appSecret,
		AuthInfo{},
	}

	_, err := vc.GetToken()
	if err != nil {
		return nil, err
	}
	return vc, nil
}

const (
	OneHour = 3600 * 1000
)

//----------------------------------------Token----------------------------------------//
//获取token  返回的expiretime 秒  当过期的时候
func (vc *VivoClient) GetToken() (AuthInfo, error) {
	now := time.Now().UnixNano() / int64(time.Millisecond)
	if vc.AuthToken.token != "" && vc.AuthToken.validTime>now {
		return vc.AuthToken, nil
	}
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(vc.AppId + vc.AppKey + strconv.FormatInt(now, 10) + vc.AppSecret))
	sign := hex.EncodeToString(md5Ctx.Sum(nil))

	formData, err := json.Marshal(&AuthTokenReq{
		AppId:     vc.AppId,
		AppKey:    vc.AppKey,
		Timestamp: now,
		Sign:      sign,
	})
	if err != nil {
		return AuthInfo{}, err
	}

	req, err := http.NewRequest("POST", ProductionHost+AuthURL, bytes.NewReader(formData))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return AuthInfo{}, err
	}
	res, err := handleResponse(resp)
	if err != nil {
		return AuthInfo{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return AuthInfo{}, errors.New("HTTP status code:"+strconv.Itoa(resp.StatusCode))
	}

	var result TokenResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return AuthInfo{}, err
	}
	if result.Result != 0 {
		return AuthInfo{}, errors.New(fmt.Sprintf("%s", res))
	}

	vc.AuthToken.token = result.AuthToken
	vc.AuthToken.validTime = now + OneHour
	return vc.AuthToken, nil
}

//----------------------------------------Sender----------------------------------------//
// 根据regID，发送消息到指定设备上
func (vc *VivoClient) Send(msg *Message, regID string) (*SendResult, error) {
	params := vc.assembleSendParams(msg, regID)
	res, err := vc.doPost(ProductionHost+SendURL, params)
	if err != nil {
		return nil, err
	}
	var result SendResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// 保存群推消息公共体接口
func (vc *VivoClient) SaveListPayload(msg *MessagePayload) (*SendResult, error) {
	res, err := vc.doPost(ProductionHost+SaveListPayloadURL, msg.JSON())
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
func (vc *VivoClient) SendList(msg *MessagePayload, regIds []string) (*SendResult, error) {
	if len(regIds) < 2 || len(regIds) > 1000 {
		return nil, errors.New("regIds个数必须大于等于2,小于等于 1000")
	}
	saveResult, err := vc.SaveListPayload(msg)
	if err != nil {
		return nil, err
	}
	// save中已经判断过saveResult的code了
	bytes, err := json.Marshal(NewListMessage(regIds, saveResult.TaskId))
	if err != nil {
		return nil, err
	}

	//推送
	pushRes, err := vc.doPost(ProductionHost+PushToListURL, bytes)
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
func (vc *VivoClient) SendAll(msg *MessagePayload) (*SendResult, error) {
	res, err := vc.doPost(ProductionHost+PushToAllURL, msg.JSON())
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
func (vc *VivoClient) GetMessageStatusByJobKey(jobKey string) (*BatchStatusResult, error) {
	params := vc.assembleStatusByJobKeyParams(jobKey)
	res, err := vc.doGet(ProductionHost+MessagesStatusURL, params)
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

func (vc *VivoClient) assembleSendParams(msg *Message, regID string) []byte {
	msg.RegId = regID
	jsondata := msg.JSON()
	return jsondata
}

func (vc *VivoClient) assembleStatusByJobKeyParams(jobKey string) string {
	form := url.Values{}
	form.Add("taskIds", jobKey)
	return "?" + form.Encode()
}






// HTTP



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

func (vc *VivoClient) doPost(url string, formData []byte) ([]byte, error) {
	var result []byte
	var req *http.Request
	var resp *http.Response
	var err error

	req, err = http.NewRequest("POST", url, bytes.NewReader(formData))
	req.Header.Set("Content-Type", "application/json")
	authInfo, err := vc.GetToken()
	if err != nil {
		return result, err
	}
	req.Header.Set("authToken", authInfo.token)

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

func (vc *VivoClient) doGet(url string, params string) ([]byte, error) {
	var result []byte
	var req *http.Request
	var resp *http.Response
	var err error
	req, err = http.NewRequest("GET", url+params, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authToken", vc.AuthToken.token)

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
