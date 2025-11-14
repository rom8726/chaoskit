package exporters

import (
	"net/http"
)

// PrometheusHandler returns an HTTP handler for the /metrics endpoint
func (p *PrometheusExporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	metrics := p.Export()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(metrics))
}

// Handler returns an http.Handler for use with standard library
func (p *PrometheusExporter) Handler() http.Handler {
	return http.HandlerFunc(p.ServeHTTP)
}
