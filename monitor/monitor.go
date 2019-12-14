// Package monitor identifies new backups as they become available. It watches a
// given backup directory, determining when a backup completes using a small
// state machine.
package monitor

import (
	"fmt"
	"strings"

	"github.com/fsnotify/fsnotify"
)

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
		return nil, fmt.Errorf("failed to create watcher: %v", err)
	}
	if err = watcher.Add(dir); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to watch %v: %v", dir, err)
	}
	return &Monitor{
		watcher: watcher,
		Backups: filter(watcher.Events),
		Errors:  watcher.Errors,
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
		var lastCreated string
		for event := range events {
			switch state {
			// looking for a create file event
			case 0:
				if event.Op != fsnotify.Create {
					continue
				}
				lastCreated = event.Name
				state = 1
			// observing writes; waiting for one to the meta file
			case 1:
				if event.Op != fsnotify.Write {
					// reset
					state = 0
					continue
				}
				if strings.HasSuffix(event.Name, ".unf") {
					// new backup file is being written; we see >5000 of
					// these events before it finishes
					continue
				}
				if strings.HasSuffix(event.Name, ".json") {
					// meta file is being written, which means backup file
					// is complete, so we can put it on the channel and
					// reset
					complete <- lastCreated
					state = 0
				}
			}
		}
	}()
	return complete
}
