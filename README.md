# UniFi Backup

[![CI](https://github.com/gebn/unifibackup/actions/workflows/ci.yaml/badge.svg)](https://github.com/gebn/unifibackup/actions/workflows/ci.yaml)
[![Docker Hub](https://img.shields.io/docker/pulls/gebn/unifibackup.svg)](https://hub.docker.com/r/gebn/unifibackup)
[![Go Reference](https://pkg.go.dev/badge/github.com/gebn/unifibackup/v2.svg)](https://pkg.go.dev/github.com/gebn/unifibackup/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/gebn/unifibackup)](https://goreportcard.com/report/github.com/gebn/unifibackup)

A daemon that monitors a UniFi Controller's `autobackup` directory, uploading new backups to S3 as soon as they are ready.

## Controller Setup

Enable auto backup under `Settings > Auto Backup` on your controller. Set the occurrence to as often as your internet connection can take - the more frequent, the less data you are likely to lose. You only need to retain one (i.e. the latest) backup, however you may want to keep around more; this daemon will work correctly regardless, as it only pays attention to new files.

## Systemd Setup

The executable is intended to run under systemd. The following instructions detail how to set up the service.

1. Download the [latest](https://github.com/gebn/unifibackup/releases/latest) archive to `/opt/unifibackup`.

2. Copy `unifibackup.service` into `/etc/systemd/system`, and open the file in your favourite editor.
   1. Change the bucket to the one you want to upload to (see *IAM Policy* below for required permissions), and optionally override the destination prefix.
   2. Set `AWS_REGION` to the region of the bucket above.
   3. If not using an instance profile, and credentials are not configured elsewhere, set the `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables.

3. Execute the following as `root`:

       systemctl enable unifibackup.service        # start on boot
       systemctl start unifibackup.service         # start right now
       systemctl status unifibackup.service        # check running smoothly (look for "active (running)")

Running the final `status` command again after a few multiples of the backup frequency should show the daemon successfully uploading new files:

    $ sudo systemctl status unifibackup.service
    ● unifibackup.service - A utility to upload Unifi Controller backups to S3
         Loaded: loaded (/etc/systemd/system/unifibackup.service; enabled; vendor preset: enabled)
         Active: active (running) since Tue 2022-07-05 22:44:45 UTC; 3 days ago
           Docs: https://github.com/gebn/unifibackup/blob/master/README.md
       Main PID: 54658 (unifibackup)
          Tasks: 8 (limit: 1030)
         Memory: 17.2M
            CPU: 6min 49.127s
         CGroup: /system.slice/unifibackup.service
                 └─54658 /opt/unifibackup/unifibackup --bucket unifi-backups-euw2 --timeout 5s

    Jul 09 12:26:10 i-01371f14862e87305 unifibackup[54658]: uploaded autobackup_7.1.66_20220709_1225_1657369500081.unf (18.991 MiB) in 513ms
    Jul 09 13:26:11 i-01371f14862e87305 unifibackup[54658]: uploaded autobackup_7.1.66_20220709_1325_1657373100075.unf (19.063 MiB) in 522ms
    Jul 09 14:26:13 i-01371f14862e87305 unifibackup[54658]: uploaded autobackup_7.1.66_20220709_1425_1657376700067.unf (19.139 MiB) in 497ms
    Jul 09 15:26:12 i-01371f14862e87305 unifibackup[54658]: uploaded autobackup_7.1.66_20220709_1525_1657380300075.unf (19.232 MiB) in 469ms
    Jul 09 16:26:25 i-01371f14862e87305 unifibackup[54658]: uploaded autobackup_7.1.66_20220709_1625_1657383900071.unf (19.221 MiB) in 421ms
    Jul 09 17:26:16 i-01371f14862e87305 unifibackup[54658]: uploaded autobackup_7.1.66_20220709_1725_1657387500070.unf (19.221 MiB) in 566ms
    Jul 09 18:26:09 i-01371f14862e87305 unifibackup[54658]: uploaded autobackup_7.1.66_20220709_1825_1657391100088.unf (19.241 MiB) in 594ms
    Jul 09 19:26:10 i-01371f14862e87305 unifibackup[54658]: uploaded autobackup_7.1.66_20220709_1925_1657394700100.unf (19.220 MiB) in 540ms
    Jul 09 20:26:26 i-01371f14862e87305 unifibackup[54658]: uploaded autobackup_7.1.66_20220709_2025_1657398300071.unf (19.216 MiB) in 598ms
    Jul 09 21:26:15 i-01371f14862e87305 unifibackup[54658]: uploaded autobackup_7.1.66_20220709_2125_1657401900050.unf (19.194 MiB) in 510ms

### IAM Policy

Regardless of how the daemon runs, it requires put and delete permissions on the destination bucket. This can be achieved with the following IAM policy:

    {
        "Version": "2012-10-17",
        "Statement": [
            {
                "Effect": "Allow",
                "Action": [
                    "s3:PutObject",
                    "s3:DeleteObject"
                ],
                "Resource": "arn:aws:s3:::<bucket>/<prefix>*"
            }
        ]
    }

If running in EC2, an [instance profile](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use_switch-role-ec2.html) is the best way to permission this.

## Alerting

The binary exposes Prometheus metrics at `:9184/metrics` by default.
To be alerted when a backup has not succeeded in 3 hours, the following rule can be used:

```yaml
groups:
- name: alerting:unifi
  rules:
  - alert: StaleUniFiBackup
    expr: |2
        time() - unifibackup_last_success_time_seconds > 3 * 60 * 60
      and
        unifibackup_last_success_time_seconds > 0
    annotations:
      description: 'UniFi Controller {{ $labels.instance }} not backed up successfully for {{ humanizeDuration $value }}'
```

## Restore

When listing backups available for restore, the UniFi software only consults an `autobackup_meta.json` file in the autobackup directory.
As this meta file can be recreated using only the underlying backups themselves, we do not back it up.
Instead, a `genmeta` subcommand exists to create an appropriate meta file given an autobackup directory containing _only_ backup files.
It is envisaged that the controller provisioner will download the latest backup from S3 to the autobackup directory, and run `unifibackup genmeta` to create `autobackup_meta.json`.
The backup can then be correctly listed and restored via the UI.
Note the meta file must be readable and writable by the `unifi` user, otherwise it will not be updated by the controller, which is relied on by the daemon for determining when new backups have finished being written.
