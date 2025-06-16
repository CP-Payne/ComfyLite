package comfy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/CP-Payne/comfylite/internal/tracker"
	"github.com/gorilla/websocket"
)

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type promptRequest struct {
	Prompt   json.RawMessage `json:"prompt"`
	ClientID string          `json:"client_id"`
}

type promptResponse struct {
	PromptID   string         `json:"prompt_id"`
	Number     int            `json:"number"`
	NodeErrors map[string]any `json:"node_errors"`
}

type Client interface {
	Start(ctx context.Context, eventChan chan<- tracker.Event) error
	Submit(workflow []byte) (string, error)
}

type client struct {
	baseURL    string
	clientID   string
	httpClient *http.Client
	conn       *websocket.Conn
}

func NewClient(baseURL, clientID string) Client {
	return &client{
		baseURL:    baseURL,
		clientID:   clientID,
		httpClient: &http.Client{},
	}
}

func (c *client) Start(ctx context.Context, eventChan chan<- tracker.Event) error {

	u, _ := url.Parse(c.baseURL)
	wsURL := fmt.Sprintf("ws://%s/ws?clientId=%s", u.Host, c.clientID)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}

	c.conn = conn

	go c.dispatcher(ctx, eventChan)

	return nil
}

func (c *client) dispatcher(_ context.Context, eventChan chan<- tracker.Event) {
	defer c.conn.Close()

	for {
		msgType, rawMsg, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("Failed reading message: %v", err)
			continue
		}

		switch msgType {
		case websocket.TextMessage:
			var comfyEvent Event
			err := json.Unmarshal(rawMsg, &comfyEvent)
			if err != nil {
				log.Printf("Error: failed to unmarshal comfy event: %v", err)
				continue
			}

			dataMap, ok := comfyEvent.Data.(map[string]interface{})
			if !ok {
				log.Printf("Warn: data field missing or not a map")
				continue
			}

			promptID, ok := dataMap["prompt_id"].(string)
			if !ok {
				// Not all events contain prompt_id, ignore those for now
				// log.Printf("Warn: prompt_id field missing or not a string")
				continue
			}

			var internalEvent tracker.Event

			switch comfyEvent.Type {
			case "execution_start":
				internalEvent = tracker.Event{Type: tracker.EventExecutionStart, PromptID: promptID}
			case "execution_success":
				internalEvent = tracker.Event{Type: tracker.EventExecutionFinished, PromptID: promptID}

				// TODO: Add event types for "execution_interupted" and determin if there is a type for errors
			}

			eventChan <- internalEvent

		case websocket.BinaryMessage:
			eventChan <- tracker.Event{Type: tracker.EventImageReceived, Data: rawMsg}
		default:
			log.Println("Unknown message type: ", msgType)

		}

	}
}

func (c *client) Submit(workflow []byte) (string, error) {

	reqPayload := promptRequest{
		Prompt:   json.RawMessage(workflow),
		ClientID: c.clientID,
	}

	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal prompt request: %w", err)
	}

	endpoint := c.baseURL + "/prompt"
	resp, err := c.httpClient.Post(endpoint, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to submit prompt: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-200 status from ComfyUI: %s", resp.Status)
	}

	var promptResp promptResponse
	if err := json.NewDecoder(resp.Body).Decode(&promptResp); err != nil {
		return "", fmt.Errorf("failed to decode prompt response: %w", err)
	}

	if len(promptResp.NodeErrors) != 0 {
		fmt.Printf("WARN: node errors is not nil: %v\n", promptResp.NodeErrors)
	}

	return promptResp.PromptID, nil
}
