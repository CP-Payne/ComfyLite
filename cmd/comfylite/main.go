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
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using default values or environment variables.")
	}

	comfyLiteAddr := GetEnvOrDefault("COMFYLITE_ADDRESS", ":8083")
	comfyUIAddr := GetEnvOrDefault("COMFYUI_ADDRESS", "http://127.0.0.1:8000")

	ctx := context.Background()
	clientID := uuid.New()

	manager := workflow.NewManager("templates", "configs")
	webhookNotifier := notifier.NewHTTPNotifier()

	comfyClient := comfy.NewClient(comfyUIAddr, clientID.String())
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

	log.Printf("Starting ComfyLite server on %s\n", comfyLiteAddr)
	if err := http.ListenAndServe(comfyLiteAddr, r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

}

func GetEnvOrDefault(key string, defaultVal string) string {

	val, exist := os.LookupEnv(key)
	if !exist {
		return defaultVal
	}

	return val
}
