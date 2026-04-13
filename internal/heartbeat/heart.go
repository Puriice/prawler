package heartbeat

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/url"
	"time"
)

type Heart struct {
	uuid     string
	endpoint url.URL
	interval time.Duration

	httpClient http.Client

	maxBackoff     time.Duration
	currentBackoff time.Duration
}

func acquireUUID(client http.Client, endpoint url.URL) (string, error) {
	resp, err := client.Get(endpoint.JoinPath("/api/v1/heart").String())

	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var payload HeartbeatPayload

	err = json.NewDecoder(resp.Body).Decode(&payload)

	if err != nil {
		return "", err
	}

	return payload.UUID, nil
}

func NewHeart(endpoint url.URL, interval *time.Duration) *Heart {
	if interval == nil {
		inter := 2 * time.Second
		interval = &inter
	}

	client := http.Client{
		Timeout: 2 * time.Second,
	}

	uuid, err := acquireUUID(client, endpoint)

	if err != nil {
		return nil
	}

	return &Heart{
		uuid:     uuid,
		endpoint: endpoint,
		interval: *interval,

		httpClient: client,

		maxBackoff:     30 * time.Second,
		currentBackoff: *interval,
	}
}

func (h *Heart) SetInterval(interval time.Duration, maxBackoff time.Duration) {
	h.interval = interval
	h.currentBackoff = interval
	h.maxBackoff = maxBackoff
}

func (h *Heart) sendHeartbeat() error {
	payload := HeartbeatPayload{UUID: h.uuid}
	data, _ := json.Marshal(payload)

	resp, err := h.httpClient.Post(h.endpoint.JoinPath("/api/v1/heartbeat").String(), "application/json", bytes.NewBuffer(data))

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

		h.currentBackoff = h.interval

		time.Sleep(h.interval)
	}
}

func (h *Heart) UUID() string {
	return h.uuid
}
