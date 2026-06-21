package mailer

import (
	"context"
	"crypto/tls"
	"io"

	gomail "github.com/wneessen/go-mail"
)

// CalendarMethod is the iTIP method embedded in a text/calendar part.
type CalendarMethod string

// iTIP method constants for CalendarPart.
const (
	CalendarMethodRequest CalendarMethod = "REQUEST"
	CalendarMethodCancel  CalendarMethod = "CANCEL"
	CalendarMethodReply   CalendarMethod = "REPLY"
)

// Part is a single MIME body part.
type Part struct {
	ContentType string
	Content     string
}

// TextPart returns a text/plain Part.
func TextPart(content string) Part {
	return Part{ContentType: "text/plain", Content: content}
}

// HTMLPart returns a text/html Part.
func HTMLPart(content string) Part {
	return Part{ContentType: "text/html", Content: content}
}

// CalendarPart returns a text/calendar Part. Method defaults to REQUEST.
func CalendarPart(ics string, method CalendarMethod) Part {
	if method == "" {
		method = CalendarMethodRequest
	}
	return Part{ContentType: "text/calendar; method=" + string(method), Content: ics}
}

// Attachment is a file attached to the email (multipart/mixed).
type Attachment struct {
	Name   string
	Reader io.Reader
}

// Embedded is an inline resource referenced as cid:Name in HTML (multipart/related).
type Embedded struct {
	Name   string
	Reader io.Reader
}

// Message is a single outbound email.
type Message struct {
	To          []string
	CC          []string
	BCC         []string
	From        string // overrides Config.From when set
	FromName    string // overrides Config.FromName when set
	ReplyTo     string
	Subject     string
	Parts       []Part
	Attachments []Attachment
	Embedded    []Embedded
}

// Service manages a go-mail client and its lifecycle.
type Service struct {
	config Config
	client *gomail.Client
}

// New returns an uninitiated Service. Connection is deferred to OnStart via fx.Module.
func New(cfg Config) *Service {
	return &Service{config: cfg}
}

func (s *Service) connect(ctx context.Context) error {
	opts := []gomail.Option{
		gomail.WithPort(s.config.Port),
		gomail.WithTimeout(s.config.Timeout),
	}

	if s.config.Username != "" {
		opts = append(
			opts,
			gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
			gomail.WithUsername(s.config.Username),
			gomail.WithPassword(s.config.Password),
		)
	}

	switch {
	case s.config.TLSEnabled:
		opts = append(opts, gomail.WithSSL())
	case s.config.StartTLS:
		opts = append(opts, gomail.WithTLSPortPolicy(gomail.TLSMandatory))
	default:
		opts = append(opts, gomail.WithTLSPortPolicy(gomail.NoTLS))
	}

	if s.config.Insecure {
		opts = append(opts, gomail.WithTLSConfig(&tls.Config{InsecureSkipVerify: true})) //nolint:gosec // user-controlled via config
	}

	client, err := gomail.NewClient(s.config.Host, opts...)
	if err != nil {
		return Domain.Mark(err, ErrConnFailed) //nolint:wrapcheck // Domain.Mark is the wrapping layer
	}

	if err := client.DialWithContext(ctx); err != nil {
		return Domain.Mark(err, ErrConnFailed) //nolint:wrapcheck // Domain.Mark is the wrapping layer
	}
	if err := client.Close(); err != nil {
		return Domain.Wrap(err, "close test connection")
	}

	s.client = client
	return nil
}

// Send delivers msg via SMTP.
func (s *Service) Send(ctx context.Context, msg Message) error {
	if len(msg.Parts) == 0 {
		return Domain.New("message has no parts") //nolint:wrapcheck // Domain.New is the error origin
	}

	from := s.config.From
	if msg.From != "" {
		from = msg.From
	}
	if from == "" {
		return Domain.New("no from address: set Config.From or Message.From") //nolint:wrapcheck // Domain.New is the error origin
	}
	fromName := s.config.FromName
	if msg.FromName != "" {
		fromName = msg.FromName
	}

	m := gomail.NewMsg()
	if err := m.FromFormat(fromName, from); err != nil {
		return Domain.Wrap(err, "set from")
	}
	if err := m.To(msg.To...); err != nil {
		return Domain.Wrap(err, "set to")
	}
	if len(msg.CC) > 0 {
		if err := m.Cc(msg.CC...); err != nil {
			return Domain.Wrap(err, "set cc")
		}
	}
	if len(msg.BCC) > 0 {
		if err := m.Bcc(msg.BCC...); err != nil {
			return Domain.Wrap(err, "set bcc")
		}
	}
	if msg.ReplyTo != "" {
		if err := m.ReplyTo(msg.ReplyTo); err != nil {
			return Domain.Wrap(err, "set reply-to")
		}
	}
	m.Subject(msg.Subject)

	for i, p := range msg.Parts {
		ct := gomail.ContentType(p.ContentType)
		if i == 0 {
			m.SetBodyString(ct, p.Content)
		} else {
			m.AddAlternativeString(ct, p.Content)
		}
	}

	for _, a := range msg.Attachments {
		if err := m.AttachReader(a.Name, a.Reader); err != nil {
			return Domain.Wrapf(err, "attach %s", a.Name)
		}
	}

	for _, e := range msg.Embedded {
		if err := m.EmbedReader(e.Name, e.Reader); err != nil {
			return Domain.Wrapf(err, "embed %s", e.Name)
		}
	}

	if err := s.client.DialAndSendWithContext(ctx, m); err != nil {
		return Domain.Mark(err, ErrSendFailed) //nolint:wrapcheck // Domain.Mark is the wrapping layer
	}
	return nil
}

// Close is a no-op; go-mail dials per send.
func (s *Service) Close() error {
	return nil
}

// NewConnected creates and immediately connects a Service. Use in tests and CLIs.
func NewConnected(ctx context.Context, cfg Config) (*Service, error) {
	svc := New(cfg)
	if err := svc.connect(ctx); err != nil {
		return nil, err
	}
	return svc, nil
}
