package status

type CrawlerStatus string

const (
	Activation   CrawlerStatus = "ACTIVATE"
	Heartbeat    CrawlerStatus = "HEARTBEAT"
	Deactivation CrawlerStatus = "DEACTIVATE"
)

var (
	ValidCrawlerStatus = []CrawlerStatus{Activation, Heartbeat, Deactivation}
)
