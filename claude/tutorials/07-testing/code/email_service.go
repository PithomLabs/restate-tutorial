package main

import (
	restate "github.com/restatedev/sdk-go"
)

type EmailService struct{}

// SendVerificationEmail sends verification email
func (EmailService) SendVerificationEmail(
	ctx restate.Context,
	email string,
	token string,
) error {
	_, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		ctx.Log().Info("Sending verification email",
			"email", email,
			"token", token)

		// In real app: call SendGrid, SES, etc.
		// For now, just log

		return true, nil
	})

	return err
}
