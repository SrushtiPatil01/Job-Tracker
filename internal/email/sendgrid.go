package email

import (
	"fmt"
	"log"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// Sender sends reminder emails through SendGrid.
type Sender struct {
	apiKey    string
	fromEmail string
	toEmail   string
}

// NewSender creates a new email sender.
func NewSender(apiKey, fromEmail, toEmail string) *Sender {
	return &Sender{
		apiKey:    apiKey,
		fromEmail: fromEmail,
		toEmail:   toEmail,
	}
}

// SendReminder sends a follow-up reminder email for a specific application.
func (s *Sender) SendReminder(company, role, applicationID string) error {
	from := mail.NewEmail("Job Tracker", s.fromEmail)
	to := mail.NewEmail("", s.toEmail)
	subject := fmt.Sprintf("Reminder: Follow up on %s — %s", company, role)

	body := fmt.Sprintf(
		"Hey,\n\nIt's been 7+ days since you applied to %s for the %s role (ID: %s).\n\n"+
			"Time to follow up or update the status if you've heard back.\n\n"+
			"— Job Tracker",
		company, role, applicationID,
	)

	message := mail.NewSingleEmail(from, subject, to, body, "")
	client := sendgrid.NewSendClient(s.apiKey)

	resp, err := client.Send(message)
	if err != nil {
		return fmt.Errorf("sendgrid send failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("sendgrid returned status %d: %s", resp.StatusCode, resp.Body)
	}

	log.Printf("Reminder email sent for application %s (%s at %s), status: %d", applicationID, role, company, resp.StatusCode)
	return nil
}