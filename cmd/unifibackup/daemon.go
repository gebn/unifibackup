package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gebn/unifibackup/v2/cmd/unifibackup/monitor"
	"github.com/gebn/unifibackup/v2/cmd/unifibackup/uploader"
)

var (
	lastSuccessTime = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "unifibackup_last_success_time_seconds",
		Help: "When the last successful backup completed, as seconds since the Unix Epoch.",
	})
	failures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "unifibackup_failures_total",
		Help: "The number of end-to-end upload operations that have not completed successfully.",
	})
	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "unifibackup_request_duration_seconds",
			Help: "The time taken to execute the handlers of web server endpoints.",
		},
		[]string{"path"},
	)
)

// daemon waits for new backups, uploading them as they finish being written.
func daemon(ctx context.Context, metricsListen, backupDir string, uploader *uploader.Uploader, timeout time.Duration) error {
	log.SetFlags(0) // systemd already prefixes logs with the timestamp

	wg := sync.WaitGroup{}
	defer wg.Wait()

	srv := buildServer(metricsListen)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("server did not close cleanly: %v", err)
		}
	}()
	defer func() {
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("failed to close listener: %v", err)
		}
	}()

	monitor, err := monitor.New(backupDir)
	if err != nil {
		return err
	}
	defer func() {
		if err := monitor.Close(); err != nil {
			log.Printf("failed to close monitor: %v", err)
		}
	}()

	log.Printf("waiting for new backups")
	for {
		select {
		case path := <-monitor.Backups:
			ctx, cancel := context.WithTimeout(ctx, timeout)
			if _, err := uploader.Upload(ctx, path); err != nil {
				log.Printf("upload error: %v", err)
				failures.Inc()
			} else {
				lastSuccessTime.SetToCurrentTime()
			}
			cancel()
		case err := <-monitor.Errors:
			return fmt.Errorf("monitor error: %w", err)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func buildServer(listen string) *http.Server {
	registerHandler("/metrics", promhttp.Handler())
	return &http.Server{
		Addr:              listen,
		ReadHeaderTimeout: time.Second * 5,
		IdleTimeout:       time.Minute * 3,
	}
}

// registerHandler adds an instrumented version of the provided handler to the
// default mux at the indicated path.
func registerHandler(path string, handler http.Handler) {
	http.Handle(path, promhttp.InstrumentHandlerDuration(
		requestDuration.MustCurryWith(prometheus.Labels{
			"path": path,
		}),
		handler,
	))
}
