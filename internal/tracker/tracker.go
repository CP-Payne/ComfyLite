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
				t.finalizePrompt(t.currentPrompt, "channel closed")
				return
			}

			// Drain the remaining timeout if it has not stopped successfully
			// Note: any event received by this eventChan is a sign of life (heartbeat) (such as the progress and executing event)
			// and should reset the timer
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
			t.finalizePrompt(t.currentPrompt, "tracker timed out")
		}
	}
}

func (t *tracker) processEvent(event Event) {
	t.promptsMux.Lock()
	defer t.promptsMux.Unlock()

	switch event.Type {
	case EventExecutionStart:

		if t.currentPrompt != nil {
			t.finalizePrompt(t.currentPrompt, "new prompt started")
		}

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
		t.tryFinalizeOnSuccess(t.currentPrompt)
	case EventExecutionFinished:
		if t.currentPrompt == nil {
			log.Println("Warning: Received finished event with no active prompt.")
			return
		}
		t.currentPrompt.ExecutionFinished = true
		t.tryFinalizeOnSuccess(t.currentPrompt)

	}
}

func (t *tracker) tryFinalizeOnSuccess(prompt *PromptState) {
	if prompt == nil {
		return
	}

	if prompt.ExecutionFinished && len(prompt.ImagesReceived) == prompt.ImagesExpected {
		t.finalizePrompt(prompt, "all images and finished signal received")
	}
}

func (t *tracker) finalizePrompt(prompt *PromptState, reason string) {
	if _, exist := t.allPrompts[prompt.ID]; !exist {
		return
	}

	log.Printf("Finalizing prompt. Reason: %s\n", reason)

	var payload notifier.WebhookPayload

	if prompt.ExecutionFinished && len(prompt.ImagesReceived) == prompt.ImagesExpected {
		log.Printf("Prompt %s finished successfully.", prompt.ID)

		// Encode binary images before sending
		var encodedImages []string
		for _, image := range prompt.ImagesReceived {
			encodedImages = append(encodedImages, base64.StdEncoding.EncodeToString(image))
		}

		payload = notifier.WebhookPayload{
			Status:   "success",
			PromptID: prompt.ID,
			Images:   encodedImages,
		}

		// Still send the result to the channel for incase any consumers wants to wait for the success result
		prompt.ResultChan <- &Result{Success: true, Images: prompt.ImagesReceived}
	} else {
		err := fmt.Errorf("prompt failed validation: expected %d images, got %d. finish_signal: %t", prompt.ImagesExpected, len(prompt.ImagesReceived), prompt.ExecutionFinished)
		log.Printf("Prompt %s failed: %v", prompt.ID, err)

		payload = notifier.WebhookPayload{
			Status:   "failure",
			PromptID: prompt.ID,
			Error:    err.Error(),
		}

		prompt.ResultChan <- &Result{Success: false, Error: err}
	}

	if prompt.WebhookURL != "" {
		if err := t.notifier.Notify(prompt.WebhookURL, payload); err != nil {
			fmt.Printf("Error queueing webhook notification for prompt %s: %v\n", prompt.ID, err)
		}
	}

	close(prompt.ResultChan)
	delete(t.allPrompts, prompt.ID)

	if t.currentPrompt != nil && t.currentPrompt.ID == prompt.ID {
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
