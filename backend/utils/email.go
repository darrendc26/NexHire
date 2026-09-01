package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/resend/resend-go/v3"
)

type Service struct {
	client *resend.Client
	from   string
}

func NewService() *Service {
	return &Service{
		client: resend.NewClient(os.Getenv("RESEND_API_KEY")),
		from:   os.Getenv("EMAIL_FROM"),
	}
}

func (s *Service) SendOTP(
	ctx context.Context,
	to string,
	otp string,
) error {
	if os.Getenv("RESEND_API_KEY") == "" {
		return fmt.Errorf("RESEND_API_KEY is not set")
	}
	if s.from == "" {
		return fmt.Errorf("EMAIL_FROM is not set")
	}

	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: "Your NexHire verification code",
		Html: fmt.Sprintf(`
			<h2>NexHire Email Verification</h2>
			<p>Your verification code is:</p>
			<h1>%s</h1>
			<p>This code expires in 5 minutes.</p>
			<p>If you didn't request this code, you can safely ignore this email.</p>
		`, otp),
	}

	_, err := s.client.Emails.SendWithContext(ctx, params)
	return err
}
