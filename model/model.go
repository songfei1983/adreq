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

type Imp struct {
	ID            string  `json:"id"`
	Banner        *Banner `json:"banner,omitempty"`
	Video         *Video  `json:"video,omitempty"`
	BidFloor      float64 `json:"bidfloor"`
	FloorCurrency string  `json:"floor_currency,omitempty"`
	Secure        int     `json:"secure,omitempty"`
	Ext           ImpExt  `json:"ext,omitempty"`
}

type ImpExt struct {
	PMP *PMP `json:"pmp,omitempty"`
}

type PMP struct {
	PrivateAuction int    `json:"private_auction,omitempty"`
	Deals          []Deal `json:"deals,omitempty"`
}

type Deal struct {
	ID       string  `json:"id"`
	BidFloor float64 `json:"bid_floor"`
}

type Banner struct {
	W      int      `json:"w"`
	H      int      `json:"h"`
	Format []Format `json:"format,omitempty"`
}

type Format struct {
	W int `json:"w"`
	H int `json:"h"`
}

type Video struct {
	W     int      `json:"w"`
	H     int      `json:"h"`
	Mimes []string `json:"mimes"`
}

type Site struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Page string `json:"page,omitempty"`
}

type App struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Bundle string `json:"bundle,omitempty"`
}

type Device struct {
	IDFA string `json:"ifa,omitempty"`
	IP   string `json:"ip,omitempty"`
	UA   string `json:"ua,omitempty"`
}

type User struct {
	ID   string     `json:"id,omitempty"`
	Data []UserData `json:"data,omitempty"`
}

type UserData struct {
	ID      string    `json:"id"`
	Segment []Segment `json:"segment,omitempty"`
}

type Segment struct {
	ID string `json:"id"`
}

type BidResponse struct {
	ID      string    `json:"id"`
	SeatBid []SeatBid `json:"seatbid"`
}

type SeatBid struct {
	Seat string `json:"seat,omitempty"`
	Bid  []Bid  `json:"bid"`
}

type Bid struct {
	ID       string  `json:"id"`
	ImpID    string  `json:"impid"`
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
	adm      string  `json:"adm,omitempty"`
	Ext      BidExt  `json:"ext,omitempty"`
}

type BidExt struct {
	DealID     string `json:"deal_id,omitempty"`
	Priority   int    `json:"priority,omitempty"`
	CampaignID string `json:"campaign_id,omitempty"`
}
