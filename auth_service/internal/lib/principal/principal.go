package principal

import (
	"context"
	"uuid"

	"github.com/mikhaeris/sky-bank/auth_service/internal/apperr"
	"google.golang.org/grpc/metadata"
)

var ErrUnauthenticated = apperr.New(apperr.Unauthenticated, "AUTHENTICATION_REQUIRED", "authentication required")

const (
	identityIDMetadata       = "x-identity-id"
	emailMetadata            = "x-email"
	metadataSingleValueCount = 1
	metadataFirstValueIndex  = 0
)

type Principal struct {
	IdentityID uuid.UUID
	Email      string
}

func PrincipalFromContext(ctx context.Context) (Principal, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return Principal{}, ErrUnauthenticated
	}

	identityIDs := md.Get(identityIDMetadata)
	if len(identityIDs) != metadataSingleValueCount {
		return Principal{}, ErrUnauthenticated
	}

	identityID, err := uuid.Parse(identityIDs[metadataFirstValueIndex])
	if err != nil {
		return Principal{}, ErrUnauthenticated
	}

	emails := md.Get(emailMetadata)
	if len(emails) != metadataSingleValueCount {
		return Principal{}, ErrUnauthenticated
	}

	return Principal{
		IdentityID: identityID,
		Email:      emails[metadataFirstValueIndex],
	}, nil
}
