package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/CP-Payne/comfylite/internal/service"
)

const workflowName = "starter"

type Handler struct {
	service service.Service
}

func NewHandler(service service.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) HandleGenerateImage(w http.ResponseWriter, r *http.Request) {

	var genRequest GenerationRequest
	err := json.NewDecoder(r.Body).Decode(&genRequest)
	if err != nil {
		log.Printf("failed to decode request body")
		return
	}

	var response GenerateResponse

	// IMPORTANT: the keys in the prompt must match those specified in the config/<workflow>.yaml
	promptParams := make(map[string]any)

	if genRequest.Prompt == "" {

		response.Error = "prompt cannot be empty"

		respData, err := json.Marshal(response)
		if err != nil {
			fmt.Printf("failed to marshal response data: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		if _, err = w.Write(respData); err != nil {
			fmt.Printf("failed to write response body to writer: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	promptParams["prompt"] = genRequest.Prompt

	if genRequest.Height <= 0 {
		promptParams["height"] = 450
	}
	promptParams["height"] = genRequest.Height

	if genRequest.Width <= 0 {
		promptParams["width"] = 450
	}
	promptParams["width"] = genRequest.Height

	if genRequest.ImageCount <= 0 {
		promptParams["imageCount"] = 1
	}

	promptParams["imageCount"] = genRequest.ImageCount

	seed := time.Now().UnixNano()
	promptParams["seed"] = seed

	result, err := h.service.GenerateImage(r.Context(), workflowName, promptParams, genRequest.WebhookURL)
	if err != nil || result.PromptID == "" {
		fmt.Printf("failed to generate image: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response.PromptID = result.PromptID

	respData, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("failed to marshal response data: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(respData)

}
