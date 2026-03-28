package master

type EventType string

const (
	CrawlerSpawn      EventType = "prawler.master.spawn"
	CrawlerHeartBeats EventType = "prawler.master.heartbeat"
	CrawlerDie        EventType = "prawler.master.die"
)

var ValidEventType = []EventType{
	CrawlerSpawn,
	CrawlerHeartBeats,
	CrawlerDie,
}
