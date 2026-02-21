// Package mail provides a fluent SMTP mailer for Kashvi.
//
// Usage:
//
//	mail.To("user@example.com").
//	    Subject("Welcome to Kashvi!").
//	    Body("<h1>Hello</h1>").
//	    Send()
//
//	// With template
//	mail.To("user@example.com").
//	    Subject("Invoice").
//	    Template("invoice.html", data).
//	    Send()
package mail

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	"github.com/shashiranjanraj/kashvi/config"
)

// ------------------- Config -------------------

// SMTP holds connection credentials (populated from env/config).
type SMTP struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	FromName string
}

func defaultSMTP() SMTP {
	return SMTP{
		Host:     config.Get("MAIL_HOST", "smtp.mailtrap.io"),
		Port:     config.Get("MAIL_PORT", "587"),
		Username: config.Get("MAIL_USERNAME", ""),
		Password: config.Get("MAIL_PASSWORD", ""),
		From:     config.Get("MAIL_FROM", "hello@kashvi.app"),
		FromName: config.Get("MAIL_FROM_NAME", "Kashvi"),
	}
}

// ------------------- Message -------------------

// Message is a fluent builder for an email.
type Message struct {
	to          []string
	cc          []string
	bcc         []string
	subject     string
	body        string
	isHTML      bool
	attachments []attachment
	smtpCfg     SMTP
}

type attachment struct {
	name    string
	content []byte
}

// To sets the primary recipients.
func To(addresses ...string) *Message {
	return &Message{
		to:      addresses,
		isHTML:  true,
		smtpCfg: defaultSMTP(),
	}
}

// CC adds CC recipients.
func (m *Message) CC(addresses ...string) *Message {
	m.cc = append(m.cc, addresses...)
	return m
}

// BCC adds BCC recipients.
func (m *Message) BCC(addresses ...string) *Message {
	m.bcc = append(m.bcc, addresses...)
	return m
}

// Subject sets the email subject.
func (m *Message) Subject(s string) *Message {
	m.subject = s
	return m
}

// Body sets the email body (HTML by default).
func (m *Message) Body(html string) *Message {
	m.body = html
	m.isHTML = true
	return m
}

// Text sets a plain-text body.
func (m *Message) Text(text string) *Message {
	m.body = text
	m.isHTML = false
	return m
}

// Template renders an html/template file with data and sets it as the body.
// templatePath is relative to your templates directory.
func (m *Message) Template(templatePath string, data interface{}) *Message {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		m.body = fmt.Sprintf("<!-- template error: %v -->", err)
		return m
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		m.body = fmt.Sprintf("<!-- render error: %v -->", err)
		return m
	}
	m.body = buf.String()
	m.isHTML = true
	return m
}

// Attach adds a file attachment (in-memory).
func (m *Message) Attach(name string, content []byte) *Message {
	m.attachments = append(m.attachments, attachment{name: name, content: content})
	return m
}

// UseConfig overrides the SMTP settings for this message.
func (m *Message) UseConfig(cfg SMTP) *Message {
	m.smtpCfg = cfg
	return m
}

// ------------------- Sending -------------------

// Send delivers the email via SMTP.
func (m *Message) Send() error {
	cfg := m.smtpCfg
	if cfg.Username == "" {
		return fmt.Errorf("mail: MAIL_USERNAME not configured")
	}

	from := fmt.Sprintf("%s <%s>", cfg.FromName, cfg.From)
	allTo := append(m.to, append(m.cc, m.bcc...)...)

	raw := m.buildRaw(from)

	addr := cfg.Host + ":" + cfg.Port
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	// Use TLS for port 465, STARTTLS for 587/25.
	if cfg.Port == "465" {
		return m.sendTLS(addr, auth, cfg.From, allTo, raw, cfg.Host)
	}
	return smtp.SendMail(addr, auth, cfg.From, allTo, raw)
}

func (m *Message) sendTLS(addr string, auth smtp.Auth, from string, to []string, raw []byte, host string) error {
	tlsCfg := &tls.Config{ServerName: host}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("mail: TLS dial: %w", err)
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Quit()

	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = w.Write(raw)
	return err
}

func (m *Message) buildRaw(from string) []byte {
	contentType := "text/plain"
	if m.isHTML {
		contentType = "text/html"
	}

	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(m.to, ", ") + "\r\n")
	if len(m.cc) > 0 {
		b.WriteString("Cc: " + strings.Join(m.cc, ", ") + "\r\n")
	}
	b.WriteString("Subject: " + m.subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString(fmt.Sprintf("Content-Type: %s; charset=\"UTF-8\"\r\n", contentType))
	b.WriteString("\r\n")
	b.WriteString(m.body)
	return []byte(b.String())
}
