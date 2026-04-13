package master

type EventType string

const (
	URIRegister EventType = "prawler.master.uri"
)

var ValidEventType = []EventType{
	URIRegister,
}
