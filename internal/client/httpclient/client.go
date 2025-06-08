package httpclient

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/CP-Payne/comfylite/internal/models"
)

type Client struct {
	BaseURL    string
	Webhook    string
	HTTPClient *http.Client
}

// TODO: Add optional proxy
func New(baseURL string, webhook string) *Client {

	return &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		Webhook:    webhook,
	}
}

func (c *Client) GenerateImage(reqData models.GenerateRequest) (*models.GenerateResponse, error) {
	url := c.BaseURL + "/prompt"

	data, err := json.Marshal(reqData)
	if err != nil {
		log.Fatal("failed to marshal GenerateRequest")
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		log.Fatalf("Error creating request")
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		log.Printf("Error sending request to %s: %v\n", c.BaseURL+"/prompt", err)
		return nil, err
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v\n", err)
		return nil, err
	}

	log.Printf("Response body: %s\n", string(respBody))

	return nil, nil
}

func (c *Client) NotifyWebhook(reqData models.WebhookRequestData) error {
	data, err := json.Marshal(reqData)
	if err != nil {
		log.Fatal("Failed to marshal webhook request data")
		return err
	}

	req, err := http.NewRequest("POST", c.Webhook, bytes.NewBuffer(data))
	if err != nil {
		log.Fatalf("error creating request")
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		log.Printf("Error sending request to %s: %v\n", c.Webhook, err)
		return err
	}
	defer resp.Body.Close()

	return nil
}
