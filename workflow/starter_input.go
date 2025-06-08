package workflow

import "encoding/json"

type StarterPromptNodeInputs struct {
	Text string          `json:"text"`
	Clip json.RawMessage `json:"clip"`
}
