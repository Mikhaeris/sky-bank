package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/lib/pq"
	"github.com/mikhaeris/sky-bank/customer_service/internal/domain"
)

var (
	ErrCustomerNotFound = errors.New("customer not found")
	ErrRowAlreadyExists = errors.New("customer already exists")
)

type CustomerRepository struct {
	DB *sql.DB
}

func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{
		DB: db,
	}
}

func (cr *CustomerRepository) GetCustomerByID(ctx context.Context, identityID uuid.UUID) (domain.Customer, error) {
	query := `
		SELECT id, email, email_verified,
		first_name, last_name, middle_name, birth_date, gender,
		kyc_status,
		created_at, updated_at
		FROM customers
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var customer domain.Customer

	err := cr.DB.QueryRowContext(ctx, query, identityID).Scan(
		&customer.ID,
		&customer.Email,
		&customer.EmailVerified,
		&customer.FirstName,
		&customer.LastName,
		&customer.MiddleName,
		&customer.BirthDate,
		&customer.Gender,
		&customer.KycStatus,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Customer{}, ErrCustomerNotFound
		}

		return domain.Customer{}, fmt.Errorf("get customer by id: %w", err)
	}

	return customer, nil
}

func (cr *CustomerRepository) Insert(ctx context.Context, dto domain.Customer) (domain.Customer, error) {
	query := `
		INSERT INTO customers (
			id,
			email,
			email_verified,
			first_name,
			last_name,
			middle_name,
			birth_date,
			gender,
			kyc_status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	args := []any{
		dto.ID,
		dto.Email,
		dto.EmailVerified,
		dto.FirstName,
		dto.LastName,
		dto.MiddleName,
		dto.BirthDate,
		dto.Gender,
		dto.KycStatus,
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := cr.DB.ExecContext(ctx, query, args...)
	if err != nil {
		var pqErr *pq.Error

		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return domain.Customer{}, ErrRowAlreadyExists
		}

		return domain.Customer{}, err
	}

	return dto, nil
}

func (cr *CustomerRepository) Update(ctx context.Context, customer domain.Customer) error {
	query := `
		UPDATE customers
		SET
			email = $1,
			email_verified = $2,
			first_name = $3,
			last_name = $4,
			middle_name = $5,
			birth_date = $6,
			gender = $7,
			kyc_status = $8,
			created_at = $9,
			updated_at = $10
		WHERE id = $11 AND updated_at = $12`

	args := []any{
		customer.Email,
		customer.EmailVerified,
		customer.FirstName,
		customer.LastName,
		customer.MiddleName,
		customer.BirthDate,
		customer.Gender,
		customer.KycStatus,
		customer.CreatedAt,
		time.Now(),
		customer.ID,
		customer.UpdatedAt,
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	res, err := cr.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	count, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if count == 0 {
		return fmt.Errorf("not rows affected")
	}

	return nil
}
