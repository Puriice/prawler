package master

type EventType string

const (
	EventURIRegister  EventType = "prawler.master.uri"
	EventCrawlConfirm EventType = "prawler.master.confirm"
)

var ValidEventType = []EventType{
	EventURIRegister,
	EventCrawlConfirm,
}
