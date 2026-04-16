package frontier

type EventType string

const (
	EventURIRegister  EventType = "prawler.frontier.uri"
	EventCrawlConfirm EventType = "prawler.frontier.confirm"
)

var ValidEventType = []EventType{
	EventURIRegister,
	EventCrawlConfirm,
}
