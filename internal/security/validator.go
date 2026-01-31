package security

// WebhookURLValidator validates webhook URLs for SSRF and HTTPS enforcement
type WebhookURLValidator struct {
	allowLocalhost bool
}

// NewWebhookURLValidator creates a new webhook URL validator
// If allowLocalhost is true, HTTP URLs for localhost are permitted (development mode)
func NewWebhookURLValidator(allowLocalhost bool) *WebhookURLValidator {
	return &WebhookURLValidator{
		allowLocalhost: allowLocalhost,
	}
}

// ValidateWebhookURL validates a webhook URL
// Returns nil if the URL is valid, or an error describing the issue
func (v *WebhookURLValidator) ValidateWebhookURL(url string) error {
	return ValidateWebhookURL(url, v.allowLocalhost)
}
