package domain

import "time"

type Tokens struct {
	Access    string
	Refresh   string
	ExpiresAt time.Time
}

type TokensDTO struct {
	Refersh string
}
