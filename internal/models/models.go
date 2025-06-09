package models

import (
	"github.com/CP-Payne/comfylite/workflow"
	"github.com/google/uuid"
)

type GenerateRequest struct {
	Prompt   map[string]*workflow.Node `json:"prompt"`
	ClientID uuid.UUID                 `json:"client_id"`
}

type GenerateResponse struct {
	PromptID string `json:"prompt_id"`
}

type WebhookRequestData struct {
	PromptID string   `json:"prompt_id"`
	Images   []string `json:"images"`
}
