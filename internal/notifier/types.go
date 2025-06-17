package notifier

type WebhookPayload struct {
	Status   string   `json:"status"`
	PromptID string   `json:"prompt_id"`
	Images   []string `json:"images"`
	Error    string   `json:"error,omitempty"`
}
