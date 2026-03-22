package model

type PMP struct {
	PrivateAuction int    `json:"private_auction,omitempty"`
	Deals          []Deal `json:"deals,omitempty"`
}
