package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gebn/unifibackup/monitor"
	"github.com/gebn/unifibackup/uploader"

	"github.com/alecthomas/kingpin"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gebn/go-stamp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	help      = "Watches for new UniFi Controller backups and uploads them to S3."
	namespace = "unifibackup"
)

var (
	backupDir = kingpin.Flag("dir", "Path of the autobackup directory.").
			Default("/var/lib/unifi/backup/autobackup").
			String() // we don't use ExistingDir() as that requires a valid dir to use `--version`
	bucket = kingpin.Flag("bucket", "Name of the S3 bucket to upload to.").
		Required().
		String()
	prefix = kingpin.Flag("prefix", "Prepended to the file name to form the object key of backups.").
		Default("unifi/").
		String()
	metrics = kingpin.Flag("metrics", "A listen spec on which to expose Prometheus metrics.").
		String()
	timeout = kingpin.Flag("timeout", "The amount of time to allow put and delete requests to S3 to complete.").
		Default("5m").
		Duration()

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
	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "request_duration_seconds",
			Help: "The time taken to execute the handlers of web server " +
				"endpoints.",
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
				cancel()
				return fmt.Errorf("upload error: %v", err)
			}
			cancel()
			lastSuccessTime.SetToCurrentTime()
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
	log.SetFlags(0) // systemd already prefixes logs with the timestamp

	kingpin.CommandLine.Help = help
	kingpin.Version(stamp.Summary())
	kingpin.Parse()

	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt)
	done := make(chan struct{})
	go func() {
		<-sigs
		close(done)
	}()

	if *metrics != "" {
		srv := buildServer(*metrics)
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

	monitor, err := monitor.New(*backupDir)
	if err != nil {
		log.Fatal(err)
	}

	sess := session.Must(session.NewSession())
	svc := s3.New(sess)
	uploader := uploader.New(svc, *bucket, *prefix)
	if err = backupLoop(uploader, monitor, *timeout, done); err != nil {
		log.Println(err)
	}

	if err = monitor.Close(); err != nil {
		log.Printf("failed to close monitor: %v", err)
	}
}
