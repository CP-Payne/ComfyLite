package tracker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type Tracker interface {
	Start(ctx context.Context, eventChan <-chan Event)
	Subscribe(promptID string, imagesExpected int) (<-chan *Result, error)
}

type tracker struct {
	currentPrompt *PromptState
	allPrompts    map[string]*PromptState
	promptsMux    sync.RWMutex
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

			timeout.Stop()
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

	if isSuccess {
		log.Printf("Prompt %s finished successfully.", promptToValidate.ID)
		promptToValidate.ResultChan <- &Result{Success: true, Images: promptToValidate.ImagesReceived}
	} else {

		log.Printf("Prompt %s failed: %v", promptToValidate.ID, validationError)
		promptToValidate.ResultChan <- &Result{Success: false, Error: validationError}
	}

	close(promptToValidate.ResultChan)
	delete(t.allPrompts, promptToValidate.ID)

	if !isShutdownOrTimeout {
		t.currentPrompt = nil
	}
}

func (t *tracker) Subscribe(promptID string, imagesExpected int) (<-chan *Result, error) {
	t.promptsMux.Lock()
	defer t.promptsMux.Unlock()

	if _, exists := t.allPrompts[promptID]; exists {
		return nil, fmt.Errorf("prompt ID %s is already being tracked", promptID)
	}

	newState := &PromptState{
		ImagesExpected: imagesExpected,
		ImagesReceived: make([][]byte, 0, imagesExpected),
		ResultChan:     make(chan *Result, 1),
	}

	t.allPrompts[promptID] = newState

	return newState.ResultChan, nil
}
