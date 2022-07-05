// Package autobackup generates an autobackup_meta.json from backups.
//
// This is necessary because the UniFi Controller does not list the autobackup
// directory, it only reads the contents of the inventory file. A missing
// inventory results in UniFi stating no backups are available.
//
// Non-auto backups are not handled. These will have been downloaded to the
// client machine, and backup files in the backup/ directory are not found by
// the controller. We could be missing an equivalent meta JSON file that the
// controller is looking for.
package autobackup
