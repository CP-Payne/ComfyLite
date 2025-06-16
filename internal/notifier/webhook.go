package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Notifier interface {
	Notify(webhookURL string, payload WebhookPayload) error
}

type httpNotifier struct {
	client *http.Client
}

func NewHTTPNotifier() Notifier {
	return &httpNotifier{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (n *httpNotifier) Notify(webhookURL string, payload WebhookPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	go func() {
		resp, err := n.client.Post(webhookURL, "application/json", bytes.NewBuffer(body))
		if err != nil {
			fmt.Printf("Error sending webhook to %s: %v\n", webhookURL, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			fmt.Printf("Received non-2xx status from webhook %s: %s\n", webhookURL, resp.Status)
		} else {
			fmt.Printf("Successfully sent webhook for prompt %s\n", payload.PromptID)
		}
	}()

	return nil
}
