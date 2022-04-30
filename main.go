package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gebn/unifibackup/v2/monitor"
	"github.com/gebn/unifibackup/v2/uploader"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gebn/go-stamp/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace = "unifibackup"
)

var (
	buildInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "build_info",
			Help:      "The version and commit of the software. Constant 1.",
		},
		// the runtime version is already exposed by the default Go collector
		[]string{"version", "commit"},
	)
	buildTime = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "build_time_seconds",
		Help:      "When the software was built, as seconds since the Unix Epoch.",
	})
	lastSuccessTime = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "last_success_time_seconds",
		Help:      "When the last successful backup completed, as seconds since the Unix Epoch.",
	})
	failures = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "failures_total",
		Help:      "The number of end-to-end upload operations that have not completed successfully.",
	})
	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "request_duration_seconds",
			Help:      "The time taken to execute the handlers of web server endpoints.",
		},
		[]string{"path"},
	)
)

func backupLoop(uploader *uploader.Uploader, monitor *monitor.Monitor, timeout time.Duration, done <-chan struct{}) error {
	for {
		select {
		case path := <-monitor.Backups:
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			if _, err := uploader.Upload(ctx, path); err != nil {
				log.Printf("upload error: %v", err)
				failures.Inc()
			} else {
				lastSuccessTime.SetToCurrentTime()
			}
			cancel()
		case err := <-monitor.Errors:
			return fmt.Errorf("monitor error: %v", err)
		case <-done:
			return nil
		}
	}
}

func init() {
	buildInfo.WithLabelValues(stamp.Version, stamp.Commit).Set(1)
	buildTime.Set(float64(stamp.Time().UnixNano()) / float64(time.Second))
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

func main() {
	if err := app(context.Background()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func app(ctx context.Context) error {
	flgBackupDir := flag.String("dir", "/var/lib/unifi/backup/autobackup", "Path of the autobackup directory.")
	flgBucket := flag.String("bucket", "", "Name of the S3 bucket to upload to.")
	flgPrefix := flag.String("prefix", "unifi/", "Prepended to the backup file name to form the object key.")
	flgMetrics := flag.String("metrics", "", "A listen spec on which to expose Prometheus metrics. If empty, no metrics are exposed.")
	flgTimeout := flag.Duration("timeout", 5*time.Minute, "The amount of time to allow for put and delete S3 requests.")
	flgVersion := flag.Bool("version", false, "Print program version and exit.")
	flag.Parse()

	if *flgVersion {
		fmt.Println(stamp.Summary())
		return nil
	}

	log.SetFlags(0) // systemd already prefixes logs with the timestamp

	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt)
	done := make(chan struct{})
	go func() {
		<-sigs
		close(done)
	}()

	if *flgMetrics != "" {
		srv := buildServer(*flgMetrics)
		wg := sync.WaitGroup{}
		defer wg.Wait()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := srv.ListenAndServe(); err != http.ErrServerClosed {
				log.Printf("server did not close cleanly: %v", err)
			}
		}()
		defer func() {
			if err := srv.Shutdown(context.Background()); err != nil {
				log.Printf("failed to close listener: %v", err)
			}
		}()
	}

	monitor, err := monitor.New(*flgBackupDir)
	if err != nil {
		return err
	}
	defer func() {
		if err := monitor.Close(); err != nil {
			log.Printf("failed to close monitor: %v", err)
		}
	}()

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return fmt.Errorf("failed to initialise AWS SDK: %w", err)
	}
	s3client := s3.NewFromConfig(cfg)
	uploader := uploader.New(s3client, *flgBucket, *flgPrefix)
	return backupLoop(uploader, monitor, *flgTimeout, done)
}
