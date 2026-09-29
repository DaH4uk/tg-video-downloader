package logrus

import (
	"github.com/sirupsen/logrus"

	"tg-video-downloader/internal/infrastructure/logger/redact"
)

// redactingFormatter scrubs secrets from the fully formatted entry, so they are
// removed from the message and from every field (e.g. errors carrying request URLs).
type redactingFormatter struct {
	next logrus.Formatter
}

func (f redactingFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	b, err := f.next.Format(entry)
	if err != nil {
		return nil, err
	}

	return redact.SecretsBytes(b), nil
}
