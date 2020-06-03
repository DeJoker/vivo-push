package vivopush

// 元信息
type ResultItem struct {
	Result int    `json:"result"`
	Desc   string `json:"desc"`
}


type TokenResult struct {
	ResultItem
	AuthToken string `json:"authToken"`
}

type SaveListResult struct {
	ResultItem
	TaskId  string                 `json:"taskId"`
}

type InvalidUser struct {
	Status int     `json:"status"`
	UserId string  `json:"userid"`
}

type SendResult struct {
	ResultItem
	TaskId  string                 `json:"taskId,omitempty"`
	Invalid InvalidUser `json:"invalidUser,omitempty"`
}



// 统计结构
type TaskData struct {
	TaskId  string `json:"taskId"`
	Target  int    `json:"target"`
	Send    int    `json:"send"`
	Receive int    `json:"receive"`
	Display int    `json:"display"`
	Click   int    `json:"click"`
	Valid   int    `json:"valid"`

	TargetInActive int    `json:"targetInActive"`
	TargetInvalid int    `json:"targetInvalid"`
	TargetUnsubscribe int    `json:"targetUnSub"`
	TargetOffline int    `json:"targetOffline"`
	Invalid int    `json:"targetInvalid"`

	Controlled int    `json:"controlled"`
	Covered int    `json:"covered"`
}


type BatchStatusResult struct {
	ResultItem
	statistics []TaskData `json:"statistics"`
}


