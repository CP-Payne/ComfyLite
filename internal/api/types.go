package api

type GenerationRequest struct {
	Prompt     string `json:"prompt"`
	ImageCount int    `json:"image_count"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	WebhookURL string `json:"webhook_url"`
}

type GenerateResponse struct {
	PromptID string `json:"prompt_id"`
	Error    string `json:"error,omitempty"`
}
