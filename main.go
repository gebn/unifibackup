package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gebn/unifibackup/monitor"
	"github.com/gebn/unifibackup/uploader"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gebn/go-stamp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gopkg.in/alecthomas/kingpin.v2"
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
		Name:      "build_time",
		Help:      "When the software was built, as seconds since the Unix Epoch.",
	})
	lastSuccessTime = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "last_success_time",
		Help:      "When the last successful backup completed, as seconds since the Unix Epoch.",
	})
)

func backupLoop(uploader *uploader.Uploader, monitor *monitor.Monitor, done <-chan struct{}) error {
	for {
		select {
		case path := <-monitor.Backups:
			if _, err := uploader.Upload(path); err != nil {
				return fmt.Errorf("upload error: %v", err)
			}
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

	monitor, err := monitor.New(*backupDir)
	if err != nil {
		log.Fatal(err)
	}

	sess := session.Must(session.NewSession())
	svc := s3.New(sess)
	uploader := uploader.New(svc, *bucket, *prefix)
	if err = backupLoop(uploader, monitor, done); err != nil {
		log.Println(err)
	}

	if err = monitor.Close(); err != nil {
		log.Printf("failed to close monitor: %v", err)
	}
}
