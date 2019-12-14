package uploader

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %v for uploading: %v", path, err)
	}
	defer f.Close()

	base := filepath.Base(path)
	key := u.Prefix + base
	start := time.Now()
	uploaded, err := u.Manager.Upload(&s3manager.UploadInput{
		Bucket: &u.Bucket,
		Key:    &key,
		Body:   f,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload %v: %v", base, err)
	}
	elapsed := time.Since(start)
	log.Printf("uploaded %v in %v", base, elapsed.Round(time.Millisecond))

	if u.previousKey != "" { // delete old backup *after* uploading new one
		_, err := u.Manager.S3.DeleteObject(&s3.DeleteObjectInput{
			Bucket: &u.Bucket,
			Key:    &u.previousKey,
		})
		if err != nil {
			// too many backups is a relatively benign failure, so continue as
			// normal
			log.Printf("failed to delete %v: %v", u.previousKey, err)
		}
	}
	u.previousKey = key
	return uploaded, nil
}
