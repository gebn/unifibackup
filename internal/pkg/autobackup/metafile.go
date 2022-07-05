package autobackup

import (
	"encoding/json"
	"io/fs"
	"sort"
	"strings"
)

// MetaFile represents the contents of the autobackup_meta.json file. Backups
// are ordered by time ascending.
type MetaFile []File

func (m MetaFile) MarshalJSON() ([]byte, error) {
	// maintaining order requires writing some custom serialisation code...
	b := strings.Builder{}
	b.WriteString("{")
	for i, f := range m {
		if i != 0 {
			b.WriteString(",")
		}
		key, err := json.Marshal(f.Name())
		if err != nil {
			return nil, err
		}
		b.Write(key)
		b.WriteString(":")
		value, err := json.Marshal(f)
		if err != nil {
			return nil, err
		}
		b.Write(value)
	}
	b.WriteString("}")
	return []byte(b.String()), nil
}

// ParseBackups turns a set of files into a representation of the meta file
// that can be serialised to autobackup_meta.json file. The set must only
// contain valid .unf backup files.
func ParseBackups(backups []fs.FileInfo) (MetaFile, error) {
	files := make([]File, 0, len(backups))
	for _, backup := range backups {
		file, err := parseFileInfo(backup)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Initiated.Before(files[j].Initiated)
	})
	return files, nil
}
