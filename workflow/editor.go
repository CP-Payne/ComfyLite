package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
)

func UpdatePrompt(wf Workflow, key string, newPrompt string) error {
	nodeToUpdate := wf[key]

	var inputs StarterPromptNodeInputs
	err := json.Unmarshal(nodeToUpdate.Inputs, &inputs)
	if err != nil {
		return err
	}

	inputs.Text = newPrompt

	rawInputs, err := json.Marshal(inputs)
	if err != nil {
		return err
	}

	nodeToUpdate.Inputs = rawInputs

	return nil
}

func RandomizeSeed(wf Workflow, key string) error {

	nodeToUpdate := wf[key]

	var inputs map[string]any
	if err := json.Unmarshal(nodeToUpdate.Inputs, &inputs); err != nil {
		return err
	}

	seedVal, ok := inputs["seed"]
	if !ok {
		return fmt.Errorf("seed entry does not exist in input struct of the provided workflow: %w", errors.New("input field does not exist"))
	}

	switch seedVal.(type) {
	case float64:
		inputs["seed"] = rand.Intn(999_999_999_999)
	default:
		return fmt.Errorf("unexpected type for seed: %T", seedVal)
	}

	rawInputs, err := json.Marshal(inputs)
	if err != nil {
		return err
	}

	nodeToUpdate.Inputs = rawInputs
	return nil
}

func UpdateImageMeta(wf Workflow, key string, imageMeta ImageMeta) error {
	nodeToUpdate := wf[key]

	var inputs map[string]any
	if err := json.Unmarshal(nodeToUpdate.Inputs, &inputs); err != nil {
		return err
	}

	updateField := func(field string, newVal int) error {
		val, ok := inputs[field]
		if !ok {
			return fmt.Errorf("%s entry does not exist in input struct of the provided Node with key %s of the provided workflow", field, key)
		}
		if _, ok := val.(float64); !ok {
			return fmt.Errorf("unexpected type for %s: %T", field, val)
		}
		inputs[field] = newVal
		return nil
	}

	if err := updateField("batch_size", imageMeta.BatchSize); err != nil {
		return err
	}
	if err := updateField("width", imageMeta.Width); err != nil {
		return err
	}
	if err := updateField("height", imageMeta.Height); err != nil {
		return err
	}

	rawInputs, err := json.Marshal(inputs)
	if err != nil {
		return err
	}

	nodeToUpdate.Inputs = rawInputs

	return nil

}
