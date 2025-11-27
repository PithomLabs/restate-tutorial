package main

import (
	"fmt"
	"os"
	"time"

	restate "github.com/restatedev/sdk-go"
)

type SendGridClient struct {
	apiKey   string
	mockMode bool
}

func NewSendGridClient() *SendGridClient {
	return &SendGridClient{
		apiKey:   os.Getenv("SENDGRID_API_KEY"),
		mockMode: os.Getenv("MOCK_MODE") == "true",
	}
}

// SendEmail sends an email (called within restate.Run)
func (c *SendGridClient) SendEmail(
	ctx restate.RunContext,
	req EmailRequest,
) (EmailResponse, error) {
	ctx.Log().Info("Sending email",
		"to", req.To,
		"subject", req.Subject)

	if c.mockMode {
		return c.mockSend(req)
	}

	return c.realSend(req)
}

func (c *SendGridClient) mockSend(req EmailRequest) (EmailResponse, error) {
	// Simulate network delay
	time.Sleep(50 * time.Millisecond)

	return EmailResponse{
		MessageID: fmt.Sprintf("msg_mock_%d", time.Now().Unix()),
		Status:    "sent",
	}, nil
}

func (c *SendGridClient) realSend(req EmailRequest) (EmailResponse, error) {
	// In production, use actual SendGrid SDK:
	// from := mail.NewEmail("Shop", "orders@shop.com")
	// to := mail.NewEmail(req.To, req.To)
	// message := mail.NewSingleEmail(from, req.Subject, to, req.Body, req.Body)
	// response, err := sendgrid.NewSendClient(c.apiKey).Send(message)

	return EmailResponse{}, fmt.Errorf("real SendGrid integration not implemented - set MOCK_MODE=true")
}
