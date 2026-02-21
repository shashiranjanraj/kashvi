// Package notification provides a multi-channel notification system for Kashvi.
//
// Define a Notification:
//
//	type WelcomeNotification struct { User models.User }
//	func (n *WelcomeNotification) Via() []string { return []string{"mail", "slack"} }
//	func (n *WelcomeNotification) ToMail() notification.MailData {
//	    return notification.MailData{
//	        Subject: "Welcome!",
//	        Body:    "<h1>Hi " + n.User.Name + "</h1>",
//	    }
//	}
//	func (n *WelcomeNotification) ToSlack() notification.SlackData {
//	    return notification.SlackData{Text: "New user: " + n.User.Name}
//	}
//
// Send:
//
//	notification.Send("user@example.com", &WelcomeNotification{User: user})
package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shashiranjanraj/kashvi/pkg/logger"
	"github.com/shashiranjanraj/kashvi/pkg/mail"
)

// ------------------- Channel data structs -------------------

// MailData carries the data needed to send an email notification.
type MailData struct {
	To      string // overrides the notifiable address if set
	Subject string
	Body    string // HTML
	Text    string // plain-text fallback
}

// SlackData carries a Slack message payload.
type SlackData struct {
	WebhookURL  string // override default if set
	Text        string
	Attachments []SlackAttachment
}

// SlackAttachment is a single Slack message attachment block.
type SlackAttachment struct {
	Color  string `json:"color,omitempty"` // "good" | "warning" | "danger"
	Title  string `json:"title,omitempty"`
	Text   string `json:"text,omitempty"`
	Footer string `json:"footer,omitempty"`
}

// WebhookData carries an arbitrary JSON payload to POST to a URL.
type WebhookData struct {
	URL     string
	Payload interface{}
	Headers map[string]string
}

// DatabaseData carries the data to be stored in a notifications table.
type DatabaseData struct {
	Type    string
	Message string
	Data    interface{}
}

// ------------------- Notification interface -------------------

// Notification is the interface every notification must satisfy.
type Notification interface {
	// Via returns the list of channel names: "mail", "slack", "webhook", "database".
	Via() []string
}

// Mailable can be implemented to support the mail channel.
type Mailable interface {
	ToMail() MailData
}

// Slackable can be implemented to support the Slack channel.
type Slackable interface {
	ToSlack() SlackData
}

// Webhookable can be implemented to support the webhook channel.
type Webhookable interface {
	ToWebhook() WebhookData
}

// Databaseable can be implemented to store the notification in the DB.
type Databaseable interface {
	ToDatabase() DatabaseData
}

// ------------------- Global config -------------------

var defaultSlackWebhook string

// SetSlackWebhook sets the default Slack incoming webhook URL.
func SetSlackWebhook(url string) { defaultSlackWebhook = url }

// ------------------- Send -------------------

// Send dispatches the notification through all channels returned by Via().
// address is typically an email address used for the mail channel.
func Send(address string, n Notification) []error {
	var errs []error
	for _, channel := range n.Via() {
		if err := dispatch(address, channel, n); err != nil {
			logger.Error("notification: channel failed",
				"channel", channel, "error", err)
			errs = append(errs, err)
		}
	}
	return errs
}

// SendAsync dispatches the notification in background goroutines.
func SendAsync(address string, n Notification) {
	go func() {
		if errs := Send(address, n); len(errs) > 0 {
			for _, e := range errs {
				logger.Error("notification: async error", "error", e)
			}
		}
	}()
}

func dispatch(address, channel string, n Notification) error {
	switch channel {
	case "mail":
		m, ok := n.(Mailable)
		if !ok {
			return fmt.Errorf("notification: %T does not implement Mailable", n)
		}
		return sendMail(address, m.ToMail())

	case "slack":
		s, ok := n.(Slackable)
		if !ok {
			return fmt.Errorf("notification: %T does not implement Slackable", n)
		}
		return sendSlack(s.ToSlack())

	case "webhook":
		wh, ok := n.(Webhookable)
		if !ok {
			return fmt.Errorf("notification: %T does not implement Webhookable", n)
		}
		return sendWebhook(wh.ToWebhook())

	default:
		return fmt.Errorf("notification: unknown channel %q", channel)
	}
}

// ------------------- Mail channel -------------------

func sendMail(address string, d MailData) error {
	to := d.To
	if to == "" {
		to = address
	}

	body := d.Body
	if body == "" {
		body = d.Text
	}

	return mail.To(to).Subject(d.Subject).Body(body).Send()
}

// ------------------- Slack channel -------------------

type slackPayload struct {
	Text        string            `json:"text,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

func sendSlack(d SlackData) error {
	url := d.WebhookURL
	if url == "" {
		url = defaultSlackWebhook
	}
	if url == "" {
		return fmt.Errorf("notification: slack webhook URL not configured")
	}

	payload := slackPayload{
		Text:        d.Text,
		Attachments: d.Attachments,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("notification: slack marshal: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("notification: slack post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("notification: slack returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// ------------------- Webhook channel -------------------

func sendWebhook(d WebhookData) error {
	if d.URL == "" {
		return fmt.Errorf("notification: webhook URL is empty")
	}

	raw, err := json.Marshal(d.Payload)
	if err != nil {
		return fmt.Errorf("notification: webhook marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, d.URL, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("notification: webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range d.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("notification: webhook send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("notification: webhook returned HTTP %d", resp.StatusCode)
	}
	return nil
}
