package redact

import "regexp"

const placeholder = "<redacted>"

// Telegram bot tokens look like "<bot id>:<35-char secret>". They leak into logs
// through net/http errors that embed request URLs such as
// https://api.telegram.org/bot<token>/getUpdates.
var botTokenPattern = regexp.MustCompile(`\d+:[A-Za-z0-9_-]{30,}`)

// Secrets replaces Telegram bot tokens in s with a placeholder.
func Secrets(s string) string {
	return botTokenPattern.ReplaceAllLiteralString(s, placeholder)
}

// SecretsBytes is the []byte variant of Secrets.
func SecretsBytes(b []byte) []byte {
	return botTokenPattern.ReplaceAllLiteral(b, []byte(placeholder))
}
