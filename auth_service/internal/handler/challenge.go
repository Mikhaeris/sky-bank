package handler

import (
	"context"
	"uuid"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
	authv1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"
)

type ChallengeHandler struct {
	authv1.UnimplementedOtpServiceServer
	challengeService *service.ChallengeService
}

func NewChallengeHandler(challengeService *service.ChallengeService) *ChallengeHandler {
	return &ChallengeHandler{
		challengeService: challengeService,
	}
}

func MapOtpChannel(in authv1.OtpChannel) (domain.OtpChannel, error) {
	var channel domain.OtpChannel
	switch in {
	case authv1.OtpChannel_OTP_CHANNEL_SMS:
		channel = domain.OtpChannelSMS
	case authv1.OtpChannel_OTP_CHANNEL_EMAIL:
		channel = domain.OtpChannelEmail
	default:
		return "", ErrInvalidOTPChannel
	}
	return channel, nil
}

func MapOtpPurpose(in authv1.OtpPurpose) (domain.CodePurpose, error) {
	var purpose domain.CodePurpose
	switch in {
	case authv1.OtpPurpose_OTP_PURPOSE_EMAIL_VERIFICATION:
		purpose = domain.CodePurposeEmailVerification
	default:
		return "", ErrInvalidOTPPurpose
	}
	return purpose, nil
}

func (c *ChallengeHandler) CreateChallenge(ctx context.Context, in *authv1.CreateChallengeRequest) (*authv1.CreateChallengeResponse, error) {
	channel, err := MapOtpChannel(in.Channel)
	if err != nil {
		return nil, err
	}

	purpose, err := MapOtpPurpose(in.Purpose)
	if err != nil {
		return nil, err
	}

	dto := domain.IssueChallengeDTO{
		Destination: in.Destination,
		Channel:     channel,
		Purpose:     purpose,
	}

	challengeID, err := c.challengeService.Issue(ctx, dto)
	if err != nil {
		return nil, err
	}

	return &authv1.CreateChallengeResponse{
		ChallengeId: challengeID.String(),
	}, nil
}

func (c *ChallengeHandler) VerifyChallenge(ctx context.Context, in *authv1.VerifyChallengeRequest) (*authv1.VerifyChallengeResponse, error) {
	id, err := uuid.Parse(in.ChallengeId)
	if err != nil {
		return nil, ErrInvalidID
	}

	purpose, err := MapOtpPurpose(in.Purpose)
	if err != nil {
		return nil, err
	}

	dto := domain.ConsumeChallengeDTO{
		ChallengeID:         id,
		Code:                in.Code,
		ExpectedPurpose:     purpose,
		ExpectedDestination: &in.Destination,
	}

	_, err = c.challengeService.Consume(ctx, dto)
	if err != nil {
		return nil, err
	}

	return &authv1.VerifyChallengeResponse{
		Verified: true,
	}, nil
}
