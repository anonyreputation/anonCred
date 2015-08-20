package proto


type Event struct {
	// event type
	EventType int
	// parameters
	Params map[string]interface{}
}
