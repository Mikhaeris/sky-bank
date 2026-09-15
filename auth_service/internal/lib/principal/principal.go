package principal

import (
	"context"
	"uuid"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	identityIDMetadata = "x-identity-id"
	emailMetadata      = "x-email"
)

type Principal struct {
	IdentityID uuid.UUID
	Email      string
}

func PrincipalFromContext(ctx context.Context) (Principal, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return Principal{}, status.Error(
			codes.Unauthenticated,
			"authentication metadata missing",
		)
	}

	identityIDs := md.Get(identityIDMetadata)
	if len(identityIDs) != 1 {
		return Principal{}, status.Error(
			codes.Unauthenticated,
			"identity id missing",
		)
	}

	identityID, err := uuid.Parse(identityIDs[0])
	if err != nil {
		return Principal{}, status.Error(
			codes.Unauthenticated,
			"invalid identity id",
		)
	}

	emails := md.Get(emailMetadata)
	if len(emails) != 1 {
		return Principal{}, status.Error(
			codes.Unauthenticated,
			"email missing",
		)
	}

	return Principal{
		IdentityID: identityID,
		Email:      emails[0],
	}, nil
}
