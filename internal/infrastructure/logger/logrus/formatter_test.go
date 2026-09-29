package logrus

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestRedactingFormatter(t *testing.T) {
	const token = "123456789:AAH4k-9xYz_abcdefghijklmnopqrstuvw"

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetFormatter(redactingFormatter{next: &logrus.TextFormatter{DisableTimestamp: true}})

	log.WithField(errorKey, errors.New(`Post "https://api.telegram.org/bot`+token+`/getUpdates": EOF`)).
		Error("request to bot" + token + " failed")

	out := buf.String()
	if strings.Contains(out, token) {
		t.Fatalf("token leaked into log output: %s", out)
	}
	if strings.Count(out, "<redacted>") != 2 {
		t.Fatalf("expected token redacted in message and field, got: %s", out)
	}
}
