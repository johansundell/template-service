package types

import (
	"time"
)

type UsageLog struct {
	ID        int       `json:"id"`
	IdKey     int       `json:"id_key"`
	Status    int       `json:"status"`
	Method    string    `json:"method"`
	Error     string    `json:"error"`
	Endpoint  string    `json:"endpoint"`
	CreatedAt time.Time `json:"created_at"`
	Response  RawJSON   `json:"response"`
	Request   RawJSON   `json:"request"`
}

type RawJSON string

func (r RawJSON) MarshalJSON() ([]byte, error) {
	if len(r) == 0 {
		return []byte("null"), nil
	}
	return []byte(r), nil
}

func (r *RawJSON) UnmarshalJSON(data []byte) error {
	*r = RawJSON(data)
	return nil
}
