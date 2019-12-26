package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/gebn/unifibackup/monitor"
	"github.com/gebn/unifibackup/uploader"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gebn/go-stamp"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	help = "Watches for new UniFi Controller backups and uploads them to S3."
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

func backupLoop(uploader *uploader.Uploader, monitor *monitor.Monitor, done <-chan struct{}) error {
	for {
		select {
		case path := <-monitor.Backups:
			if _, err := uploader.Upload(path); err != nil {
				return fmt.Errorf("upload error: %v", err)
			}
		case err := <-monitor.Errors:
			return fmt.Errorf("monitor error: %v", err)
		case <-done:
			return nil
		}
	}
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
