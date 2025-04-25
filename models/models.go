package models

type InitialRequest struct {
	Name  string `json:"name"`
	RegNo string `json:"regNo"`
	Email string `json:"email"`
}

type User struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Follows []int  `json:"follows"`
}

type UserData struct {
	Users []User `json:"users"`
}

type ResponseData struct {
	UserData UserData `json:"users"`
	FindID   int      `json:"findId"`
	N        int      `json:"n"`
}

type InitialResponse struct {
	Webhook     string       `json:"webhook"`
	AccessToken string       `json:"accessToken"`
	Data        ResponseData `json:"data"`
}

type ResultPayload struct {
	RegNo   string  `json:"regNo"`
	Outcome [][]int `json:"outcome"`
}
