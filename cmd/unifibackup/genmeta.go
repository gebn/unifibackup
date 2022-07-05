package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gebn/unifibackup/v2/internal/pkg/autobackup"
)

// genmeta generates a autobackup_meta.json file for backups in the provided
// directory. This directory must only contain .unf files.
func genmeta(backupDir string) error {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return err
	}

	infos := []fs.FileInfo{}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !strings.HasSuffix(info.Name(), ".unf") {
			return fmt.Errorf("%v is not a backup", info.Name())
		}
		infos = append(infos, info)
	}

	meta, err := autobackup.ParseBackups(infos)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(backupDir, "autobackup_meta.json"))
	if err != nil {
		return err
	}
	defer f.Close()
	// unlike genuine meta files, this produces a trailing linefeed
	return json.NewEncoder(f).Encode(meta)
}
