package uploader

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gebn/unifibackup/internal/pkg/countingreader"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// mebibyteBytes is the number of bytes in a MiB.
	mebibyteBytes = 1024 * 1024

	namespace = "unifibackup"
	subsystem = "uploader"
)

var (
	uploadAttempts = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "upload_attempts_total",
		Help:      "The number of file uploads initiated.",
	})
	uploadFailures = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "upload_failures_total",
		Help:      "The number of file uploads that failed to complete successfully.",
	})
	deleteAttempts = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "delete_attempts_total",
		Help:      "The number of object deletes initiated.",
	})
	deleteFailures = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "delete_failures_total",
		Help:      "The number of object deletes that failed to complete successfully.",
	})
	uploadedBytes = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "uploaded_bytes_total",
		Help:      "The cumulative number of backup bytes uploaded.",
	})

	// we expose this as a gauge as we can reasonably expect it to increase over
	// time, plus connection speeds will vary widely.
	uploadDuration = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "last_upload_duration_seconds",
		Help:      "The time taken to upload the last backup.",
	})
)

type Uploader struct {
	Manager     *s3manager.Uploader
	Bucket      string
	Prefix      string
	previousKey string
}

func New(svc s3iface.S3API, bucket, prefix string) *Uploader {
	return &Uploader{
		Manager: s3manager.NewUploaderWithClient(svc),
		Bucket:  bucket,
		Prefix:  prefix,
	}
}

func (u *Uploader) Upload(path string) (*s3manager.UploadOutput, error) {
	uploadAttempts.Inc()
	f, err := os.Open(path)
	if err != nil {
		uploadFailures.Inc()
		return nil, fmt.Errorf("failed to open %v for uploading: %v", path, err)
	}
	defer f.Close()

	reader := countingreader.New(f)
	base := filepath.Base(path)
	key := u.Prefix + base
	start := time.Now()
	uploaded, err := u.Manager.Upload(&s3manager.UploadInput{
		Bucket: &u.Bucket,
		Key:    &key,
		Body:   reader,
	})
	if err != nil {
		uploadFailures.Inc()
		return nil, fmt.Errorf("failed to upload %v: %v", base, err)
	}
	elapsed := time.Since(start)
	uploadDuration.Set(elapsed.Seconds())
	uploadedBytes.Add(float64(reader.ReadBytes))
	mib := float64(reader.ReadBytes) / float64(mebibyteBytes)
	log.Printf("uploaded %v (%.3f MiB) in %v", base, mib, elapsed.Round(time.Millisecond))

	if u.previousKey != "" { // delete old backup *after* uploading new one
		deleteAttempts.Inc()
		_, err := u.Manager.S3.DeleteObject(&s3.DeleteObjectInput{
			Bucket: &u.Bucket,
			Key:    &u.previousKey,
		})
		if err != nil {
			// too many backups is a relatively benign failure, so continue as
			// normal
			deleteFailures.Inc()
			log.Printf("failed to delete %v: %v", u.previousKey, err)
		}
	}
	u.previousKey = key
	return uploaded, nil
}
