package model

type User struct {
	ID   string     `json:"id,omitempty"`
	Data []UserData `json:"data,omitempty"`
}
