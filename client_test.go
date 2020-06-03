package vivopush

import (
	"testing"
)

var appId string = "your appId"
var appKey string = "your appId"
var appSecret string = "your appSecret"

var msg1 *Message = NewVivoMessage("hi baby1", "hi1")

var regID1 string = "your regID"

func TestMiPush_Send(t *testing.T) {
func TestViVoPush_Send(t *testing.T) {
	client, err := NewClient(appId, appKey, appSecret)
	if err != nil {
		t.Errorf("TestViVoPush_Send create client failed :%+v\n", err)
		return
	}
	result, err := client.Send(msg1, regID1)
	if err != nil {
		t.Errorf("TestViVoPush_Send failed :%+v\n", err)
		return
	}
	t.Logf("result=%+v\n", result)
}



func TestViVoPush_SendList(t *testing.T) {
	client, err := NewClient(appId, appKey, appSecret)
	if err != nil {
		t.Errorf("TestViVoPush_Send create client failed :%+v\n", err)
		return
	}

	listmsg := NewListPayloadMessage("haha", "xixi")
	//saveRes, err := client.SaveListPayload()
	//if err != nil {
	//	t.Errorf("TestViVoPush_Send save message :%+v\n", err)
	//}


	result, err := client.SendList(listmsg, []string{regID1})
	if err != nil {
		t.Errorf("TestViVoPush_Send failed :%+v\n", err)
		return
	}
	t.Logf("result=%#v\n", result)
}



func TestViVoPush_GetMessageStatusByJobKey(t *testing.T) {
	client, err := NewClient(appId, appKey, appSecret)
	if err != nil {
		t.Errorf("TestViVoPush_GetMessageStatusByJobKey create client failed :%+v\n", err)
		return
	}
	result, err := client.GetMessageStatusByJobKey("jobId")
	if err != nil {
		t.Errorf("TestViVoPush_GetMessageStatusByJobKey failed :%+v\n", err)
		return
	}
	t.Logf("result=%#v\n", result)
}
