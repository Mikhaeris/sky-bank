package service

import (
	"context"
	"errors"
	"time"

	"github.com/mikhaeris/sky-bank/customer_service/internal/domain"
	principal "github.com/mikhaeris/sky-bank/customer_service/internal/lib"
	"github.com/mikhaeris/sky-bank/customer_service/internal/repository"
	"github.com/mikhaeris/sky-bank/pkg/kafka"
	authv1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"
)

type CustomerService struct {
	customerRepo     *repository.CustomerRepository
	producer         *kafka.Producer
	otpServiceClient authv1.OtpServiceClient
}

func NewCustomerService(
	customerRepo *repository.CustomerRepository,
	producer *kafka.Producer,
	otpServiceClient authv1.OtpServiceClient,
) *CustomerService {
	return &CustomerService{
		customerRepo:     customerRepo,
		producer:         producer,
		otpServiceClient: otpServiceClient,
	}
}

func (c *CustomerService) GetCustomer(ctx context.Context) (domain.Customer, error) {
	principal, err := principal.PrincipalFromContext(ctx)
	if err != nil {
		return domain.Customer{}, internalErr(err)
	}

	customer, err := c.customerRepo.GetCustomerByID(ctx, principal.IdentityID)
	if err != nil {
		if errors.Is(repository.ErrCustomerNotFound, err) {
			return domain.Customer{}, ErrProfileNotCompleted
		}
		return domain.Customer{}, internalErr(err)
	}

	return customer, nil
}

func (c *CustomerService) CompleteProfile(ctx context.Context, dto domain.Customer) (domain.Customer, error) {
	principal, err := principal.PrincipalFromContext(ctx)
	if err != nil {
		return domain.Customer{}, internalErr(err)
	}

	dto.ID = principal.IdentityID
	dto.KycStatus = domain.KYCStatusNotStarted

	customer, err := c.customerRepo.Insert(ctx, dto)
	if err != nil {
		if errors.Is(err, repository.ErrRowAlreadyExists) {
			return domain.Customer{}, ErrCustomerAlreadyComplete
		}
		return domain.Customer{}, internalErr(err)
	}

	return customer, nil
}

func (c *CustomerService) StartEmailVerification(ctx context.Context) (string, error) {
	principal, err := principal.PrincipalFromContext(ctx)
	if err != nil {
		return "", internalErr(err)
	}

	customer, err := c.customerRepo.GetCustomerByID(ctx, principal.IdentityID)
	if err != nil {
		return "", internalErr(err)
	}

	req := &authv1.CreateChallengeRequest{
		Destination: customer.Email,
		Channel:     authv1.OtpChannel_OTP_CHANNEL_EMAIL,
		Purpose:     authv1.OtpPurpose_OTP_PURPOSE_EMAIL_VERIFICATION,
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	res, err := c.otpServiceClient.CreateChallenge(ctx, req)
	if err != nil {
		return "", internalErr(err)
	}

	return res.ChallengeId, nil
}

func (c *CustomerService) CompleteEmailVerification(ctx context.Context, dto domain.CompleteEmailVerificationDTO) error {
	principal, err := principal.PrincipalFromContext(ctx)
	if err != nil {
		return internalErr(err)
	}

	customer, err := c.customerRepo.GetCustomerByID(ctx, principal.IdentityID)
	if err != nil {
		return internalErr(err)
	}

	req := &authv1.VerifyChallengeRequest{
		ChallengeId: dto.ChallengeID,
		Code:        dto.OtpCode,
		Destination: customer.Email,
		Purpose:     authv1.OtpPurpose_OTP_PURPOSE_EMAIL_VERIFICATION,
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	res, err := c.otpServiceClient.VerifyChallenge(ctx, req)
	if err != nil {
		return err
	}

	if !res.Verified {
		return ErrWrongOtpCode
	}

	customer.EmailVerified = res.Verified

	err = c.customerRepo.Update(ctx, customer)
	if err != nil {
		return internalErr(err)
	}

	return nil
}
