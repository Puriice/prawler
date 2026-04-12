package heartbeat

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"time"
)

type Heart struct {
	UUID     string
	Endpoint string
	Interval time.Duration

	maxBackoff     time.Duration
	currentBackoff time.Duration
}

func NewHeart(uuid string, endpoint string, interval *time.Duration) *Heart {
	if interval == nil {
		inter := 2 * time.Second
		interval = &inter
	}

	return &Heart{
		UUID:     uuid,
		Endpoint: endpoint,
		Interval: *interval,

		maxBackoff:     30 * time.Second,
		currentBackoff: *interval,
	}
}

func (h *Heart) sendHeartbeat() error {
	payload := HeartbeatPayload{UUID: h.UUID}
	data, _ := json.Marshal(payload)

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Post(h.Endpoint+"/heartbeat", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return err
	}

	return nil
}

// Add jitter: randomize delay slightly
func addJitter(d time.Duration) time.Duration {
	jitter := time.Duration(rand.Int63n(int64(d / 2)))
	return d + jitter
}

func (h *Heart) Beat() {
	for {
		err := h.sendHeartbeat()

		if err != nil {
			// Wait with exponential backoff + jitter
			wait := addJitter(h.currentBackoff)

			time.Sleep(wait)

			// Exponential increase
			h.currentBackoff *= 2

			if h.currentBackoff > h.maxBackoff {
				h.currentBackoff = h.maxBackoff
			}

			continue
		}

		h.currentBackoff = h.Interval

		time.Sleep(h.Interval)
	}
}
