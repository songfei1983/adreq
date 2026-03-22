package model

type SeatBid struct {
	Seat string `json:"seat,omitempty"`
	Bid  []Bid  `json:"bid"`
}
