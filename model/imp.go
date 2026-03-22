package model

type Imp struct {
	ID            string  `json:"id"`
	Banner        *Banner `json:"banner,omitempty"`
	Video         *Video  `json:"video,omitempty"`
	BidFloor      float64 `json:"bidfloor"`
	FloorCurrency string  `json:"floor_currency,omitempty"`
	Secure        int     `json:"secure,omitempty"`
	Ext           ImpExt  `json:"ext,omitempty"`
}
