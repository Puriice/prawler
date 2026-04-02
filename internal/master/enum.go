package master

type EventType string

const (
	StatusReport EventType = "prawler.master.status"
)

var ValidEventType = []EventType{
	StatusReport,
}
