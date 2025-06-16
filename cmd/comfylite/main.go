package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/CP-Payne/comfylite/internal/api"
	"github.com/CP-Payne/comfylite/internal/comfy"
	"github.com/CP-Payne/comfylite/internal/notifier"
	"github.com/CP-Payne/comfylite/internal/service"
	"github.com/CP-Payne/comfylite/internal/tracker"
	"github.com/CP-Payne/comfylite/internal/workflow"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func main() {

	ctx := context.Background()
	clientID := uuid.New()

	manager := workflow.NewManager("templates", "configs")

	webhookNotifier := notifier.NewHTTPNotifier()

	comfyClient := comfy.NewClient("http://127.0.0.1:8000", clientID.String())
	eventChan := make(chan tracker.Event, 100)

	tracker := tracker.New(webhookNotifier)

	go tracker.Start(ctx, eventChan)

	if err := comfyClient.Start(ctx, eventChan); err != nil {
		log.Fatalf("Failed to start ComfyUI client: %v", err)
	}

	service := service.NewService(manager, comfyClient, tracker)

	handler := api.NewHandler(service)

	r := chi.NewRouter()
	r.Post("/generate", handler.HandleGenerateImage)

	log.Println("Starting ComfyLite server on :8083")
	if err := http.ListenAndServe(":8083", r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// TODO: Add .env

}

func GetEnvOrDefault(key string, defaultVal string) string {

	val, exist := os.LookupEnv(key)
	if !exist {
		return defaultVal
	}

	return val
}
