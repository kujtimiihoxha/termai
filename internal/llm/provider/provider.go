package provider

type Provider interface {
	NewMessage(sessionID string, content string) error
}
