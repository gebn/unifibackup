package autobackup

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"time"
)

var (
	// nameRegexp matches the name of autobackup files.
	nameRegexp = regexp.MustCompile(`autobackup_([\d\.]+)_\d{8}_\d{4}_(\d+)\.unf`)
)

// File represents a single autobackup .unf file.
type File struct {
	ControllerVersion string
	Initiated         time.Time
	SizeBytes         int64
}

// Name returns the name of the file, e.g.
// autobackup_7.1.66_20220702_2025_1656793500051.unf.
func (f File) Name() string {
	return fmt.Sprintf("autobackup_%s_%s_%v.unf",
		f.ControllerVersion,
		f.Initiated.Format("20060102_1504"),
		f.Initiated.UnixMilli())
}

// fileJSON is an ephemeral type to ease marshalling File structs to JSON.
type fileJSON struct {
	Version  string `json:"version"`
	Time     int64  `json:"time"`
	DateTime string `json:"datetime"`
	Format   string `json:"format"`
	Days     int    `json:"days"`
	Size     int64  `json:"size"`
}

func (f File) MarshalJSON() ([]byte, error) {
	return json.Marshal(fileJSON{
		Version:  f.ControllerVersion,
		Time:     f.Initiated.UnixMilli(),
		DateTime: f.Initiated.Format(time.RFC3339),
		Format:   "bson",
		Days:     -1,
		Size:     f.SizeBytes,
	})
}

// parseName extracts the controller version and initiated time from an
// autobackup file name.
func parseName(backup string) (string, time.Time, error) {
	match := nameRegexp.FindStringSubmatch(backup)
	if len(match) != 3 {
		return "", time.Time{}, fmt.Errorf("'%v' is not a valid backup name", backup)
	}
	i, err := strconv.ParseInt(match[2], 10, 64)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("'%v' is not a valid Unix millis timestamp")
	}
	return match[1], time.UnixMilli(i).In(time.UTC), nil
}

// parseFileInfo extracts the relevant data from a single autobackup file.
func parseFileInfo(info fs.FileInfo) (File, error) {
	controllerVersion, initiated, err := parseName(info.Name())
	if err != nil {
		return File{}, err
	}
	return File{
		ControllerVersion: controllerVersion,
		Initiated:         initiated,
		SizeBytes:         info.Size(),
	}, nil
}
