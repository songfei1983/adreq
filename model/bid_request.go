package model

import "time"

type BidRequest struct {
	ID        string    `json:"id"`
	Imp       []Imp     `json:"imp"`
	App       *App      `json:"app,omitempty"`
	Device    *Device   `json:"device,omitempty"`
	User      *User     `json:"user,omitempty"`
	Site      *Site     `json:"site,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
