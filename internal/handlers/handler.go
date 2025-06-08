package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/CP-Payne/comfylite/internal/client/httpclient"
	"github.com/CP-Payne/comfylite/internal/models"
	"github.com/CP-Payne/comfylite/workflow"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ImageGenHandler struct {
	logger     *zap.Logger
	httpClient *httpclient.Client
	clientID   uuid.UUID
}

type ImageGenRequest struct {
	Prompt    string `json:"prompt"`
	BatchSize int    `json:"batch_size"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type ImageGenResponse struct {
}

func NewImageGenHandler(logger *zap.Logger, httpClient *httpclient.Client, clientID uuid.UUID) *ImageGenHandler {
	return &ImageGenHandler{
		logger:     logger,
		httpClient: httpClient,
		clientID:   clientID,
	}
}

func (h *ImageGenHandler) GenerateImage(w http.ResponseWriter, r *http.Request) {

	var imageRequestInput ImageGenRequest
	err := json.NewDecoder(r.Body).Decode(&imageRequestInput)
	if err != nil {
		log.Printf("failed to decode request body")
		return
	}

	wf, err := workflow.LoadWorkflow("workflow/workflows/starter.json")
	if err != nil {
		log.Printf("Failed to load workflow: %v", err)
		return
	}

	if imageRequestInput.Prompt != "" {
		err = workflow.UpdatePrompt(wf, "6", imageRequestInput.Prompt)
		if err != nil {
			log.Printf("failed to update prompt: %v", err)
			return
		}
	}

	if imageRequestInput.BatchSize <= 0 {
		imageRequestInput.BatchSize = 1
	}

	if imageRequestInput.Height <= 0 {
		imageRequestInput.Height = 500
	}

	if imageRequestInput.Width <= 0 {
		imageRequestInput.Width = 500
	}

	err = workflow.UpdateImageMeta(wf, "5", workflow.ImageMeta{
		BatchSize: imageRequestInput.BatchSize,
		Width:     imageRequestInput.Width,
		Height:    imageRequestInput.Height,
	})

	if err != nil {
		log.Printf("failed to update image metadata: %v", err)
		return
	}

	err = workflow.RandomizeSeed(wf, "3")
	if err != nil {
		log.Printf("failed to randomize seed: %v", err)
		return
	}

	_, err = h.httpClient.GenerateImage(models.GenerateRequest{
		ClientID: h.clientID,
		Prompt:   wf,
	})

	if err != nil {
		log.Printf("failed to generate image: %v", err)
		return
	}

}
