package domain

import (
	"uuid"
)

type Identity struct {
	ID      uuid.UUID
	Email   string
	Version int
}

type IdentityDTO struct {
	Email string
}

func NewIdentity(dto IdentityDTO, identity_id uuid.UUID) *Identity {
	return &Identity{
		ID:    identity_id,
		Email: dto.Email,
	}
}
