package tracker

type EventType string

const (
	EventExecutionStart    EventType = "EXECUTION_START"
	EventExecutionFinished EventType = "EXECUTION_FINISHED"
	EventImageReceived     EventType = "IMAGE_RECEIVED"
	EventJobFailed         EventType = "JOB_FAILED"
)

type Event struct {
	Type     EventType
	PromptID string
	Data     interface{}
}

type PromptState struct {
	ID                string
	ImagesExpected    int
	ImagesReceived    [][]byte
	ExecutionFinished bool
	ResultChan        chan *Result
}

type Result struct {
	Success bool
	Images  [][]byte
	Error   error
}
