package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/CP-Payne/comfylite/internal/client/httpclient"
	"github.com/CP-Payne/comfylite/internal/client/wsclient"
	"github.com/CP-Payne/comfylite/internal/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("failed to load logger")
	}

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	serverPort := GetEnvOrDefault("PORT", "8081")
	comfyHost := GetEnvOrDefault("COMFY_HOST", "127.0.0.1")
	comfyPort := GetEnvOrDefault("COMFY_PORT", "8000")

	wsClient := wsclient.New(logger, comfyHost, comfyPort)
	clientID, err := wsClient.Connect()
	if err != nil {
		logger.Fatal("failed to connect to websocket", zap.Error(err))
		return
	}

	httpClient := httpclient.New(fmt.Sprintf("http://%s:%s", comfyHost, comfyPort), "https://webhook.site/1ec37b84-9907-408c-9cec-efca90682383")

	wsHandler := wsclient.HandleIncomingMessage(httpClient)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go wsClient.Listen(wsHandler, &wg)
	wg.Wait()

	handler := handlers.NewImageGenHandler(logger, httpClient, uuid.MustParse(clientID))

	r := chi.NewMux()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/gen", handler.GenerateImage)

	serverFullAddress := fmt.Sprintf(":%s", serverPort)

	fmt.Printf("Server listening on %s\n", serverFullAddress)
	if err := http.ListenAndServe(serverFullAddress, r); err != nil {
		log.Fatalf("Failed to start server: %s", serverFullAddress)
	}
}

func GetEnvOrDefault(key string, defaultVal string) string {

	val, exist := os.LookupEnv(key)
	if !exist {
		return defaultVal
	}

	return val
}
