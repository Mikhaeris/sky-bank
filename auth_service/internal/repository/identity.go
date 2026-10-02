package repository

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type IdentityRepository struct {
	tracer trace.Tracer
	db     DBTX
}

func NewIdentityRepository(
	tracer trace.Tracer,
	db DBTX,
) *IdentityRepository {
	return &IdentityRepository{
		tracer: tracer,
		db:     db,
	}
}

func (ir *IdentityRepository) GetByID(ctx context.Context, identityID uuid.UUID) (domain.Identity, error) {
	ctx, span := ir.tracer.Start(ctx, "identity.get")
	defer span.End()

	query := `
		SELECT id, email, version
		FROM identities
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var identity domain.Identity

	err := ir.db.QueryRow(ctx, query, identityID).Scan(
		&identity.ID,
		&identity.Email,
		&identity.Version,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Identity{}, fmt.Errorf("find identity by ID: %w", ErrRecordNotFound)
		}
		return domain.Identity{}, fmt.Errorf("find identity by ID: %w", err)
	}

	return identity, nil
}

func (ir *IdentityRepository) InsertIfAbsent(ctx context.Context, email string) error {
	ctx, span := ir.tracer.Start(ctx, "identity.insert")
	defer span.End()

	query := `
		INSERT INTO identities (id, email, version)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO NOTHING`

	id := uuid.New()

	args := []any{id, email, initialIdentityVersion}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := ir.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert identity if absent: %w", err)
	}

	return nil
}

func (ir *IdentityRepository) GetByEmail(ctx context.Context, email string) (domain.Identity, error) {
	ctx, span := ir.tracer.Start(ctx, "identity.get")
	defer span.End()

	query := `
		SELECT id, email, version
		FROM identities
		WHERE email = $1`

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var identity domain.Identity

	err := ir.db.QueryRow(ctx, query, email).Scan(
		&identity.ID,
		&identity.Email,
		&identity.Version,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Identity{}, fmt.Errorf("find identity by email: %w", ErrRecordNotFound)
		}
		return domain.Identity{}, fmt.Errorf("find identity by email: %w", err)
	}

	return identity, nil
}
