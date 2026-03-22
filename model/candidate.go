package model

type CandidateAd struct {
	ID          string
	ImpID       string
	CampaignID  string
	Price       float64
	BidFloor    float64
	Priority    int
	DealID      string
	TargetAttrs []string
	Age         int
	Gender      string
	Country     string
	Position    string
}

func (c *CandidateAd) PassesFloor() bool {
	return c.Price >= c.BidFloor
}

func (c *CandidateAd) CalculateEffectivePrice() float64 {
	return c.Price
}
