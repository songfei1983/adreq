package model

type Device struct {
	IDFA string `json:"ifa,omitempty"`
	IP   string `json:"ip,omitempty"`
	UA   string `json:"ua,omitempty"`
}
