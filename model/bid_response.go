package model

type BidResponse struct {
	ID      string    `json:"id"`
	SeatBid []SeatBid `json:"seatbid"`
}
