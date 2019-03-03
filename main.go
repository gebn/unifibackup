package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	"github.com/gebn/unifibackup/monitor"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gebn/go-stamp"
	"gopkg.in/alecthomas/kingpin.v2"
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
)

/*
uploadProcessor encapsulates the upload and delete logic. The returned
function should be called with successive completed backup file paths, which
will trigger upload followed by removal of the previous backup.
*/
func uploadProcessor(sess *session.Session) func(string) error {
	svc := s3.New(sess)
	uploader := s3manager.NewUploaderWithClient(svc)
	var previous string
	return func(path string) error {
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("Failed to open %v for uploading: %v", path, err)
		}

		base := filepath.Base(path)
		key := *prefix + base
		start := time.Now()
		_, err = uploader.Upload(&s3manager.UploadInput{
			Bucket: bucket,
			Key:    &key,
			Body:   f,
		})
		elapsed := time.Since(start)
		f.Close()
		if err != nil {
			return fmt.Errorf("Failed to upload %v: %v", base, err)
		}
		log.Printf("Uploaded %v in %v", base, elapsed.Round(time.Millisecond))

		if previous != "" { // delete old backup *after* uploading new one
			_, err := svc.DeleteObject(&s3.DeleteObjectInput{
				Bucket: bucket,
				Key:    &previous,
			})
			if err != nil {
				// too many backups is a relatively benign failure, so
				// continue as normal
				log.Printf("Failed to delete %v: %v", previous, err)
			}
		}
		previous = key
		return nil
	}
}

/*
upload sends each new backup to S3, deleting the previous one, if any.
this function provides errors on the returned channel, but waits until done is closed before stopping
*/
func upload(sess *session.Session, backups <-chan string, done <-chan struct{}, wg *sync.WaitGroup) <-chan error {
	errors := make(chan error)
	process := uploadProcessor(sess)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case path := <-backups:
				err := process(path)
				if err != nil {
					errors <- err
				}
			case <-done:
				return
			}
		}
	}()
	return errors
}

func main() {
	log.SetFlags(0) // systemd already prefixes logs with the timestamp

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

	var wg sync.WaitGroup
	sess := session.Must(session.NewSession())
	syncErrors := upload(sess, monitor.Backups, done, &wg)

	select {
	case err := <-monitor.Errors:
		log.Println("Monitor error:", err)
		close(done)
	case err := <-syncErrors:
		log.Println("Sync error:", err)
		close(done)
	case <-done:
	}

	if err = monitor.Close(); err != nil {
		log.Printf("Failed to close monitor: %v", err)
	}
	wg.Wait()
}
