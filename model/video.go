package model

type Video struct {
	W     int      `json:"w"`
	H     int      `json:"h"`
	Mimes []string `json:"mimes"`
}
