package bot

type EventType int

const (
	_                      = iota // So that the uninitialized message isn't accidentally valid
	EventConnect EventType = iota
	EventPublishMessage
)

// Events should not be modified by bots.
type Event struct {
	// Type indicates the contents of Payload, which might be nil.
	Type EventType
	// Payload is a struct defined in dispatcher, e.g., PublishMessage
	Payload interface{}
}

type Bot interface {
	Handle(e *Event)
}

type MessageType struct {
	Type string `json:"action"`
}

type PublishMessage struct {
	Data struct {
		Username        string
		Author          string
		Email           string
		Channel         string
		Gravatar        string
		Datetime        int
		Message         string
		RenderedMessage string
	}
}
