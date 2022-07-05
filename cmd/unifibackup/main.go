package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gebn/unifibackup/v2/cmd/unifibackup/uploader"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gebn/go-stamp/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spf13/cobra"
	"go.uber.org/automaxprocs/maxprocs"
)

var (
	buildInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "unifibackup_build_info",
			Help: "The version and commit of the software. Constant 1.",
		},
		// the runtime version is already exposed by the default Go collector
		[]string{"version", "commit"},
	)
	buildTime = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "unifibackup_build_time_seconds",
		Help: "When the software was built, as seconds since the Unix Epoch.",
	})
)

func main() {
	maxprocs.Set() // use the library this way to avoid logging when CPU quota is undefined
	buildInfo.WithLabelValues(stamp.Version, stamp.Commit).Set(1)
	buildTime.Set(float64(stamp.Time().UnixNano()) / float64(time.Second))

	// we can access flags via cmd, but they are untyped; in practice it's
	// easier to constrain the chaos to this function
	var (
		flgBackupDir string
		flgBucket    string
		flgPrefix    string
		flgMetrics   string
		flgTimeout   time.Duration
	)

	rootCmd := &cobra.Command{
		Use:          "unifibackup --bucket unifi-backups",
		Short:        "Copies UniFi Controller backups to S3",
		Version:      stamp.Summary(),
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			cfg, err := config.LoadDefaultConfig(ctx)
			if err != nil {
				return fmt.Errorf("failed to initialise AWS SDK: %w", err)
			}
			s3client := s3.NewFromConfig(cfg)
			uploader := uploader.New(s3client, flgBucket, flgPrefix)
			return daemon(ctx, flgMetrics, flgBackupDir, uploader, flgTimeout)
		},
	}
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.PersistentFlags().StringVar(&flgBackupDir,
		"dir",
		"/var/lib/unifi/backup/autobackup",
		"path of the UniFi autobackup directory")
	rootCmd.Flags().StringVar(&flgBucket,
		"bucket",
		"", // in hindsight, this should not have been a flag
		"name of the S3 bucket to upload to (requird)")
	rootCmd.MarkFlagRequired("bucket")
	rootCmd.Flags().StringVar(&flgPrefix,
		"prefix",
		"unifi/",
		"prepended to the backup file name to form the object key")
	rootCmd.Flags().StringVar(&flgMetrics,
		"metrics",
		":9184",
		"listen spec for the web server that exposes Prometheus metrics")
	rootCmd.Flags().DurationVar(&flgTimeout,
		"timeout",
		5*time.Minute,
		"time to allow for upload and delete requests")

	rootCmd.AddCommand(&cobra.Command{
		Use:   "genmeta",
		Short: "Generate autobackup_meta.json for the autobackup directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return genmeta(flgBackupDir)
		},
	})

	if err := rootCmd.Execute(); err != nil {
		// error already printed (SilenceErrors == false)
		os.Exit(1)
	}
}
