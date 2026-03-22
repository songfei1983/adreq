package model

type Banner struct {
	W      int      `json:"w"`
	H      int      `json:"h"`
	Format []Format `json:"format,omitempty"`
}
