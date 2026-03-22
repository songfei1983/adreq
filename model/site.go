package model

type Site struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Page string `json:"page,omitempty"`
}
