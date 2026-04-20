package events

type EmbeddingEventType string

const (
	EventEmbedding EmbeddingEventType = "prawler.embedding.id"
)

type EmbeddingEvent struct {
	Type     EmbeddingEventType `json:"event_type"`
	PageUUID string             `json:"page_uuid"`
}
