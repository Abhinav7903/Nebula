package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Abhinav7903/nebula/internal/progress"
)

type SSEWriter struct {
	hub *progress.Hub
}

func NewSSEWriter(hub *progress.Hub) *SSEWriter {
	return &SSEWriter{hub: hub}
}

func (s *SSEWriter) Stream(w http.ResponseWriter, r *http.Request, searchID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeProblem(w, http.StatusInternalServerError, "sse_error", "Streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch, done, err := s.hub.Subscribe(searchID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "sse_error", "Failed to subscribe")
		return
	}
	defer s.hub.Unsubscribe(searchID, done)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(evt)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Event, data)
			flusher.Flush()

			if evt.Event == "search_done" || evt.Event == "search_error" {
				return
			}
		}
	}
}

func (s *SSEWriter) writeEvent(w io.Writer, evt progress.Event) {
	data, _ := json.Marshal(evt)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Event, data)
}
