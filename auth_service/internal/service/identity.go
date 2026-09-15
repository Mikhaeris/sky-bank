package service

import (
	"context"
	"time"

	notificationv1 "github.com/mikhaeris/sky-bank/notification_service/api/notification/v1"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
)

func (a *AuthService) StartAuthentication(ctx context.Context, dto domain.IdentityDTO) error {
	otp, err := domain.GenerateCode(a.codeHash, dto.Email, domain.CodePurposeAuthentication, domain.CodeTTLAuthentication)
	if err != nil {
		return internalErr(err)
	}

	err = a.otpRepo.Insert(ctx, otp)
	if err != nil {
		return internalErr(err)
	}

	go func(email, otpCode string) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if err := a.otpProvider.SendOtpCode(ctx, email, otpCode); err != nil {
			a.logger.Error(
				"failed to send otp code",
				"error", err,
				"email", email,
			)
		}
	}(dto.Email, otp.CodePlaintext)

	return nil
}

func (a *AuthService) CompleteAuthentication(ctx context.Context, dto domain.OtpDto) (domain.Tokens, error) {
	code, err := a.otpRepo.GetByEmail(ctx, dto.Email, domain.CodePurposeAuthentication)
	if err != nil {
		a.logger.Error(ErrOtpCodeInvalid.Error())
		return domain.Tokens{}, ErrOtpCodeInvalid
	}

	match := a.codeHash.Verify(dto.CodePlaintext, code.CodeHash)
	if !match {
		a.logger.Error(ErrOtpCodeInvalid.Error())
		return domain.Tokens{}, ErrOtpCodeInvalid
	}

	err = a.otpRepo.DeleteByEmail(ctx, dto.Email, domain.CodePurposeAuthentication)
	if err != nil {
		return domain.Tokens{}, internalErr(err)
	}

	identity, err := a.identityRepo.GetOrCreateByEmail(ctx, dto.Email)
	if err != nil {
		return domain.Tokens{}, internalErr(err)
	}

	accessToken, err := a.jwtKey.CreateToken(&identity)
	if err != nil {
		return domain.Tokens{}, internalErr(err)
	}

	session := domain.NewSession(identity.ID, domain.RefreshTokenTTL)

	err = a.sessionRepo.Insert(ctx, *session)
	if err != nil {
		return domain.Tokens{}, internalErr(err)
	}

	go func(email string) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &notificationv1.SendNewLogInRequest{
			Email: email,
		}

		if _, err := a.notificationClient.SendNewLogIn(ctx, req); err != nil {
			a.logger.Error(
				"failed to send new log in email",
				"error", err,
			)
		}
	}(dto.Email)

	return domain.Tokens{
		Access:  accessToken,
		Refresh: session.RefreshTokenPlaintext,
	}, nil
}

func (a *AuthService) RefreshTokens(ctx context.Context, dto domain.TokensDTO) (domain.Tokens, error) {
	oldTokenHash := domain.HashRefreshToken(dto.Refersh)

	session, err := a.sessionRepo.GetByRefreshTokenHash(ctx, oldTokenHash)
	if err != nil {
		return domain.Tokens{}, ErrInvalidRefreshToken
	}

	identity, err := a.identityRepo.GetByID(ctx, session.IdentityID)
	if err != nil {
		return domain.Tokens{}, internalErr(err)
	}

	accessToken, err := a.jwtKey.CreateToken(&identity)
	if err != nil {
		return domain.Tokens{}, internalErr(err)
	}

	session.RotateRefreshToken()

	err = a.sessionRepo.Update(ctx, session, oldTokenHash)
	if err != nil {
		return domain.Tokens{}, internalErr(err)
	}

	return domain.Tokens{
		Access:  accessToken,
		Refresh: session.RefreshTokenPlaintext,
	}, nil
}
