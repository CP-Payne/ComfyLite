package workflow

import "encoding/json"

type Node struct {
	Inputs    json.RawMessage `json:"inputs"`
	ClassType string          `json:"class_type"`
	Meta      Meta            `json:"_meta"`
}

type Meta struct {
	Title string `json:"title"`
}

type Workflow map[string]*Node

type ImageMeta struct {
	BatchSize int
	Width     int
	Height    int
}
