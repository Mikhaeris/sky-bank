package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/mikhaeris/sky-bank/customer_service/internal/domain"
	"github.com/mikhaeris/sky-bank/customer_service/internal/service"
	customerv1 "github.com/mikhaeris/sky-bank/proto/gen/customer/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

type CustomerHandler struct {
	customerv1.UnimplementedCustomerServer
	customerService *service.CustomerService
}

func NewCustomerHandler(customerService *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{
		customerService: customerService,
	}
}

func (c *CustomerHandler) GetCustomer(ctx context.Context, _ *emptypb.Empty) (*customerv1.GetCustomerResponse, error) {
	customer, err := c.customerService.GetCustomer(ctx)
	if err != nil {
		return nil, err
	}

	return &customerv1.GetCustomerResponse{
		Profile: &customerv1.Profile{
			Email:      customer.Email,
			FirstName:  customer.FirstName,
			LastName:   customer.LastName,
			MiddleName: customer.MiddleName,
			BirthDate:  customer.BirthDate.String(),
			Gender:     customer.Gender,
		},
		EmailVerified: customer.EmailVerified,
		KycStatus:     customer.KycStatus,
	}, nil
}

func (c *CustomerHandler) CompleteProfile(ctx context.Context, in *customerv1.CompleteProfileRequest) (*customerv1.CompleteProfileResponse, error) {
	if in.Profile == nil {
		return &customerv1.CompleteProfileResponse{}, fmt.Errorf("invalid argument")
	}
	birthDate, err := time.Parse(domain.BirthDateLayout, in.Profile.BirthDate)
	if err != nil {
		return &customerv1.CompleteProfileResponse{}, fmt.Errorf("wrong birth date format")
	}

	dto := domain.Customer{
		Email:      in.Profile.Email,
		FirstName:  in.Profile.FirstName,
		LastName:   in.Profile.LastName,
		MiddleName: in.Profile.MiddleName,
		BirthDate:  birthDate,
		Gender:     in.Profile.Gender,
	}

	customer, err := c.customerService.CompleteProfile(ctx, dto)
	if err != nil {
		return &customerv1.CompleteProfileResponse{}, err
	}

	return &customerv1.CompleteProfileResponse{
		Profile: &customerv1.Profile{
			Email:      customer.Email,
			FirstName:  customer.FirstName,
			LastName:   customer.LastName,
			MiddleName: customer.MiddleName,
			BirthDate:  customer.BirthDate.String(),
			Gender:     customer.Gender,
		},
		EmailVerified: customer.EmailVerified,
		KycStatus:     customer.KycStatus,
	}, nil
}

func (c *CustomerHandler) StartEmailVerification(ctx context.Context, _ *emptypb.Empty) (*customerv1.StartEmailVerificationResponse, error) {
	challengeID, err := c.customerService.StartEmailVerification(ctx)
	if err != nil {
		return &customerv1.StartEmailVerificationResponse{}, err
	}

	return &customerv1.StartEmailVerificationResponse{
		ChallengeId: challengeID,
	}, nil
}

func (c *CustomerHandler) CompleteEmailVerification(ctx context.Context, in *customerv1.CompleteEmailVerificationRequest) (*emptypb.Empty, error) {
	dto := domain.CompleteEmailVerificationDTO{
		ChallengeID: in.ChallengeId,
		OtpCode:     in.OtpCode,
	}

	err := c.customerService.CompleteEmailVerification(ctx, dto)
	if err != nil {
		return &emptypb.Empty{}, err
	}

	return &emptypb.Empty{}, nil
}
