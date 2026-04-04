package model

type Bid struct {
	ID       string  `json:"id"`
	ImpID    string  `json:"impid"`
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
	adm      string
	Ext      BidExt `json:"ext,omitempty"`
}

type SpendIntent struct {
	CampaignID string
	Amount     float64
}

type CandidateDecision struct {
	Bid   *Bid
	Spend *SpendIntent
}
