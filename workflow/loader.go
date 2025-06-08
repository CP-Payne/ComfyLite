package workflow

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

func LoadWorkflow(path string) (Workflow, error) {
	var wf Workflow
	cwd, _ := os.Getwd()
	log.Println("Current working dir:", cwd)

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return wf, err
	}

	err = json.Unmarshal(data, &wf)
	return wf, err

}
