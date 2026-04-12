package master

type EventType string

const (
	StatusReport EventType = "prawler.master.status"
	URIRegister  EventType = "prawler.master.uri"
)

var ValidEventType = []EventType{
	StatusReport,
	URIRegister,
}
