package main

import (
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests.",
		},
		[]string{"path", "method", "code"},
	)
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Request latency.",
			Buckets: prometheus.DefBuckets, // 5ms .. 10s, etc.
		},
		[]string{"path", "method"},
	)
	inflight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "inflight_requests",
			Help: "Current number of in-flight requests.",
		},
	)
	itemsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "items_processed_total",
			Help: "Business counter for processed items.",
		},
		[]string{"result"},
	)
)

func instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inflight.Inc()
		start := time.Now()
		rr := &respRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rr, r)
		elapsed := time.Since(start).Seconds()

		requestDuration.WithLabelValues(r.URL.Path, r.Method).Observe(elapsed)
		httpRequestsTotal.WithLabelValues(r.URL.Path, r.Method, strconv.Itoa(rr.status)).Inc()
		inflight.Dec()
	})
}

type respRecorder struct {
	http.ResponseWriter
	status int
}

func (rr *respRecorder) WriteHeader(status int) {
	rr.status = status
	rr.ResponseWriter.WriteHeader(status)
}

func workHandler(w http.ResponseWriter, r *http.Request) {
	// Simulate some work
	n := rand.Intn(200) + 50
	time.Sleep(time.Duration(n) * time.Millisecond)

	// Random success/failure
	if rand.Float64() < 0.1 {
		itemsProcessed.WithLabelValues("error").Inc()
		http.Error(w, "oops", http.StatusInternalServerError)
		return
	}
	itemsProcessed.WithLabelValues("ok").Inc()
	w.Write([]byte("done\n"))
}

func health(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) }

func main() {
	prometheus.MustRegister(httpRequestsTotal, requestDuration, inflight, itemsProcessed)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("hello\n")) })
	mux.HandleFunc("/work", workHandler)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", health)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: instrument(mux),
	}

	log.Println("listening on :8080")
	log.Fatal(srv.ListenAndServe())
}

