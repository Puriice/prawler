package heartbeat

type Status string

const (
	Alive       Status = "Alive"
	Unconscious Status = "Unconscious"
	Dead        Status = "Dead"
)
