package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
	"github.com/mikhaeris/bank-test/auth_service/internal/models"
)

var (
	ErrDuplicateEmail = errors.New("duplicate email")
	ErrRecordNotFound = errors.New("record not found")
)

type UserRepository struct {
	DB *sql.DB
}

func NewAuthRepository(db *sql.DB) *UserRepository {
	return &UserRepository{
		DB: db,
	}
}

func (ur *UserRepository) Insert(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, activated)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	args := []any{user.ID, user.Email, user.PasswordHash, user.Activated}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var pqErr *pq.Error

	err := ur.DB.QueryRowContext(ctx, query, args...).Scan(&user.ID)
	if err != nil {
		switch {
		case errors.As(err, &pqErr) &&
			pqErr.Code == "23505" &&
			pqErr.Constraint == "users_email_key":
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func (ur *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, activated
		FROM users
		WHERE email = $1`

	var user models.User

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := ur.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Activated,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}
