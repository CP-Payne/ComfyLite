package tracker

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/CP-Payne/comfylite/internal/notifier"
)

type Tracker interface {
	Start(ctx context.Context, eventChan <-chan Event)
	Subscribe(promptID string, imagesExpected int, webhookURL string) (<-chan *Result, error)
}

type tracker struct {
	currentPrompt *PromptState
	allPrompts    map[string]*PromptState
	promptsMux    sync.RWMutex
	notifier      notifier.Notifier
}

func New(notifier notifier.Notifier) Tracker {
	return &tracker{
		allPrompts: make(map[string]*PromptState),
		notifier:   notifier,
	}
}

func (t *tracker) Start(ctx context.Context, eventChan <-chan Event) {
	log.Println("Tracker service started.")

	timeout := time.NewTimer(30 * time.Second)
	timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Tracker service shutting down.")
			return
		case event, ok := <-eventChan:
			if !ok {
				log.Println("Tracker event channel closed.")
				t.validateAndFinalizeCurrentPrompt(true)
				return
			}

			// Drain the remaining timeout if it has not stopped successfully
			if !timeout.Stop() {
				select {
				case <-timeout.C:
				default:
				}
			}
			t.processEvent(event)
			if t.currentPrompt != nil {
				timeout.Reset(30 * time.Second)
			}

		case <-timeout.C:
			log.Println("Tracker timed out waiting for new events. Finalizing last known prompt.")
			t.validateAndFinalizeCurrentPrompt(true)
		}
	}
}

func (t *tracker) processEvent(event Event) {
	t.promptsMux.Lock()
	defer t.promptsMux.Unlock()

	switch event.Type {
	case EventExecutionStart:
		t.validateAndFinalizeCurrentPrompt(false)

		prompt, ok := t.allPrompts[event.PromptID]
		if !ok {
			log.Printf("Error: Received start event for untracked prompt ID %s", event.PromptID)
			return
		}

		t.currentPrompt = prompt
		log.Printf("Tracking started for new prompt: %s", t.currentPrompt.ID)

	case EventImageReceived:
		if t.currentPrompt == nil {
			log.Println("Warning: Received binary data with no active prompt.")
			return
		}
		if binaryData, ok := event.Data.([]byte); ok {
			t.currentPrompt.ImagesReceived = append(t.currentPrompt.ImagesReceived, binaryData)
		}
	case EventExecutionFinished:
		if t.currentPrompt == nil {
			log.Println("Warning: Received finished event with no active prompt.")
			return
		}
		t.currentPrompt.ExecutionFinished = true

	}
}

func (t *tracker) validateAndFinalizeCurrentPrompt(isShutdownOrTimeout bool) {
	if t.currentPrompt == nil {
		return
	}

	promptToValidate := t.currentPrompt

	isSuccess := true
	var validationError error

	if !promptToValidate.ExecutionFinished {
		isSuccess = false
		validationError = fmt.Errorf("prompt finished without receiving an 'executed' event")
	}
	if len(promptToValidate.ImagesReceived) != promptToValidate.ImagesExpected {
		isSuccess = false
		validationError = fmt.Errorf("exected %d image but received %d", promptToValidate.ImagesExpected, len(promptToValidate.ImagesReceived))
	}

	var payload notifier.WebhookPayload
	if isSuccess {
		// TODO: store images to S3 bucket and return image URLs
		log.Printf("Prompt %s finished successfully.", promptToValidate.ID)

		// Encode binary images before sending

		var encodedImages []string
		for _, image := range promptToValidate.ImagesReceived {
			encodedImages = append(encodedImages, base64.StdEncoding.EncodeToString(image))
		}

		payload = notifier.WebhookPayload{
			Status:   "success",
			PromptID: promptToValidate.ID,
			Images:   encodedImages,
		}

		// Still send the result to the channel for incase any consumers wants to wait for the success result
		promptToValidate.ResultChan <- &Result{Success: true, Images: promptToValidate.ImagesReceived}
	} else {

		log.Printf("Prompt %s failed: %v", promptToValidate.ID, validationError)

		payload = notifier.WebhookPayload{
			Status:   "failure",
			PromptID: promptToValidate.ID,
			Error:    validationError.Error(),
		}

		promptToValidate.ResultChan <- &Result{Success: false, Error: validationError}
	}

	if promptToValidate.WebhookURL != "" {
		if err := t.notifier.Notify(promptToValidate.WebhookURL, payload); err != nil {
			fmt.Printf("Error queueing webhook notification for prompt %s: %v\n", promptToValidate.ID, err)
		}
	}

	close(promptToValidate.ResultChan)
	delete(t.allPrompts, promptToValidate.ID)

	if !isShutdownOrTimeout {
		t.currentPrompt = nil
	}
}

func (t *tracker) Subscribe(promptID string, imagesExpected int, webhookURL string) (<-chan *Result, error) {
	t.promptsMux.Lock()
	defer t.promptsMux.Unlock()

	if _, exists := t.allPrompts[promptID]; exists {
		return nil, fmt.Errorf("prompt ID %s is already being tracked", promptID)
	}

	newState := &PromptState{
		ID:             promptID,
		ImagesExpected: imagesExpected,
		ImagesReceived: make([][]byte, 0, imagesExpected),
		ResultChan:     make(chan *Result, 1),
		WebhookURL:     webhookURL,
	}

	t.allPrompts[promptID] = newState

	return newState.ResultChan, nil
}
