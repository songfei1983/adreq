package model

type UserData struct {
	ID      string    `json:"id"`
	Segment []Segment `json:"segment,omitempty"`
}
