// Package monitor identifies new backups as they become available. It watches a
// given backup directory, determining when a backup completes using a small
// state machine.
package monitor

import (
	"fmt"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "unifibackup"
	subsystem = "monitor"
)

var (
	// ops constains all possible event types we can receive from fsnotify, so
	// we can initialise all time series for the filesystem events counter.
	ops = []fsnotify.Op{
		fsnotify.Create,
		fsnotify.Write,
		fsnotify.Remove,
		fsnotify.Rename,
		fsnotify.Chmod,
	}

	eventsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "filesystem_events_total",
		Help: "The number of events received from the underlying fsnotify " +
			"library.",
	}, []string{"op"})
	errorsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "filesystem_errors_total",
		Help: "The number of errors received from the underlying fsnotify " +
			"library.",
	})
	stateGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "state",
		Help: "The current state of the monitor state machine. 0: waiting for " +
			"create file, 1: waiting for write to the meta file.",
	})
)

func init() {
	for _, op := range ops {
		eventsCounter.WithLabelValues(op.String())
	}
}

// Monitor encapsulates the output of watching for UniFi backups.
type Monitor struct {

	// watcher is the underlying handle, maintained so it can be closed.
	watcher *fsnotify.Watcher

	// Backups contains paths to newly completed backups. These paths will be
	// relative or absolute, depending on the path provided when creating the
	// monitor. Backups are safe to read immediately.
	Backups <-chan string

	// Errors contains errors returned by the underlying filesystem watcher.
	Errors <-chan error
}

// Close terminates the underlying filesystem watcher, returning any error
// encountered, or nil if the close was successful. Regardless, the monitor
// should be considered unusable.
func (m *Monitor) Close() error {
	return m.watcher.Close()
}

// New creates a new monitor that watches for and returns new UniFi backups. If
// no error occurs while setting up the watcher, the provided directory will be
// monitored until Close() is called.
func New(dir string) (*Monitor, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}
	if err = watcher.Add(dir); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to watch %v: %w", dir, err)
	}
	return &Monitor{
		watcher: watcher,
		Backups: filter(watcher.Events),
		Errors:  errors(watcher.Errors),
	}, nil
}

// filter determines when backups complete using the stream of filesystem events
// from the backup directory. Completed backups are placed on the returned
// channel.
func filter(events <-chan fsnotify.Event) <-chan string {
	complete := make(chan string)
	// we don't bother with a WaitGroup as there is never cleanup to do
	go func() {
		state := 0
		var lastBackupCreated string
		for event := range events {
			eventsCounter.WithLabelValues(event.Op.String()).Inc()
			switch state {
			// looking for a backup creation event
			case 0:
				if event.Op != fsnotify.Create || !strings.HasSuffix(event.Name, ".unf") {
					continue
				}
				lastBackupCreated = event.Name
				state = 1
				stateGauge.Set(1)
				// Observing writes; waiting for one to the meta file to indicate
				// the backup file is finished. A meta file may or may not already
				// exist.
			case 1:
				if strings.HasSuffix(event.Name, ".unf") {
					// assume new backup file is being written; we see >5000 of
					// these events before it finishes
					continue
				}
				if strings.HasSuffix(event.Name, ".json") {
					// meta file is being written, which means backup file
					// is complete, so we can put it on the channel
					complete <- lastBackupCreated
					// fall through
				}
				// something odd, or we have our backup - reset
				state = 0
				stateGauge.Set(0)
			}
		}
	}()
	return complete
}

// errors implements a no-op passthrough for the fsnotify errors channel. Its
// only purpose is to increment our errors counter.
func errors(errors <-chan error) <-chan error {
	passthrough := make(chan error)
	go func() {
		for err := range errors {
			errorsCounter.Inc()
			passthrough <- err
		}
	}()
	return passthrough
}
