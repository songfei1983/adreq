package model

type BidExt struct {
	DealID     string `json:"deal_id,omitempty"`
	Priority   int    `json:"priority,omitempty"`
	CampaignID string `json:"campaign_id,omitempty"`
}
