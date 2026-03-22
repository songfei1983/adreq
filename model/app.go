package model

type App struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Bundle string `json:"bundle,omitempty"`
}
