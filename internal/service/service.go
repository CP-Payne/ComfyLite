package service

import (
	"context"
	"fmt"

	"github.com/CP-Payne/comfylite/internal/comfy"
	"github.com/CP-Payne/comfylite/internal/tracker"
	"github.com/CP-Payne/comfylite/internal/workflow"
)

type GenerationResult struct {
	PromptID string
	Images   [][]byte
}

type Service interface {
	GenerateImage(ctx context.Context, workflowName string, params map[string]any) (*GenerationResult, error)
}

type service struct {
	workflowMgr workflow.Manager
	comfyClient comfy.Client
	tracker     tracker.Tracker
}

func NewService(wm workflow.Manager, cc comfy.Client, tracker tracker.Tracker) Service {
	return &service{
		workflowMgr: wm,
		comfyClient: cc,
		tracker:     tracker,
	}
}

func (s *service) GenerateImage(ctx context.Context, workflowName string, params map[string]any) (*GenerationResult, error) {
	finalWorkflow, err := s.workflowMgr.Build(workflowName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to build workflow: %w", err)
	}

	promptID, err := s.comfyClient.Submit(finalWorkflow)
	if err != nil {
		return nil, fmt.Errorf("failed to send workflow request: %w", err)
	}

	imageCount, ok := params["imageCount"].(int)
	if !ok {
		fmt.Println("params does not contain imageCount or is not an integer - Defaulting to 1")
		imageCount = 1
	}

	resultChan, err := s.tracker.Subscribe(promptID, imageCount)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to tracker using promptID: %s: %w", promptID, err)
	}

	select {
	case result, ok := <-resultChan:
		if !ok {
			return nil, fmt.Errorf("tracker channel closed unexpectedly for promptID: %s", promptID)
		}
		if result.Success {
			return &GenerationResult{
				PromptID: promptID,
				Images:   result.Images,
			}, nil
		}
		return nil, fmt.Errorf("prompt failed: %w", result.Error)

	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled while waiting for promptID %s: %w", promptID, ctx.Err())

	}

}
