package api

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(handler *Handler, mw *Middleware) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/search", handler.CreateSearch)
	mux.HandleFunc("GET /api/v1/search/{id}", handler.GetSearch)
	mux.HandleFunc("GET /api/v1/search/{id}/stream", handler.StreamSearch)
	mux.HandleFunc("GET /api/v1/searches", handler.ListSearches)
	mux.HandleFunc("DELETE /api/v1/search/{id}", handler.DeleteSearch)

	mux.HandleFunc("POST /api/v1/websearch", handler.WebSearch)
	mux.HandleFunc("GET /health", handler.Health)
	mux.HandleFunc("GET /ready", handler.Ready)

	mux.Handle("GET /metrics", promhttp.Handler())

	var h http.Handler = mux
	h = mw.Logging(h)
	h = mw.SecureHeaders(h)
	h = mw.RateLimit(h)
	h = mw.Auth(h)
	h = mw.Recover(h)

	return h
}
