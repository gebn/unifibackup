# UniFi Backup

[![Build Status](https://travis-ci.org/gebn/unifibackup.svg?branch=master)](https://travis-ci.org/gebn/unifibackup)
[![GoDoc](https://godoc.org/github.com/gebn/unifibackup?status.svg)](https://godoc.org/github.com/gebn/unifibackup)
[![Go Report Card](https://goreportcard.com/badge/github.com/gebn/unifibackup)](https://goreportcard.com/report/github.com/gebn/unifibackup)

A daemon that monitors a UniFi Controller's `autobackup` directory, uploading new backups to S3 as soon as they are ready.

## Controller Setup

Enable auto backup under `Settings > Auto Backup` on your controller. Set the occurrence to as often as your internet connection can take - the more frequent, the less data you are likely to lose. You only need to retain one (i.e. the latest) backup, however you may want to keep around more; this daemon will work correctly regardless, as it only pays attention to new files.

## Systemd Setup

The executable is intended to run under systemd. The following instructions detail how to set up the service.

1. Download the [latest](https://github.com/gebn/unifibackup/releases/latest) binary to `/opt/unifibackup/`, and ensure it is executable.

2. Copy `unifibackup.service` to `/etc/systemd/system`, and open the file in your favourite editor.
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
       Active: active (running) since Sat 2019-12-14 23:50:09 UTC; 1 day 20h ago
         Docs: https://github.com/gebn/unifibackup/blob/master/README.md
     Main PID: 7791 (unifibackup)
        Tasks: 11 (limit: 4915)
       CGroup: /system.slice/unifibackup.service
               └─7791 /opt/unifibackup/unifibackup --bucket bucket.example.com

    Dec 16 10:25:07 hostname unifibackup[7791]: uploaded autobackup_5.12.35_20191216_1025_1576491900006.unf in 421ms
    Dec 16 11:25:07 hostname unifibackup[7791]: uploaded autobackup_5.12.35_20191216_1125_1576495500002.unf in 435ms
    Dec 16 12:25:07 hostname unifibackup[7791]: uploaded autobackup_5.12.35_20191216_1225_1576499100011.unf in 361ms
    Dec 16 13:25:08 hostname unifibackup[7791]: uploaded autobackup_5.12.35_20191216_1325_1576502700003.unf in 396ms
    Dec 16 14:25:07 hostname unifibackup[7791]: uploaded autobackup_5.12.35_20191216_1425_1576506300006.unf in 348ms
    Dec 16 15:25:07 hostname unifibackup[7791]: uploaded autobackup_5.12.35_20191216_1525_1576509900006.unf in 397ms
    Dec 16 16:25:07 hostname unifibackup[7791]: uploaded autobackup_5.12.35_20191216_1625_1576513500006.unf in 356ms
    Dec 16 17:25:08 hostname unifibackup[7791]: uploaded autobackup_5.12.35_20191216_1725_1576517100003.unf in 409ms
    Dec 16 18:25:07 hostname unifibackup[7791]: uploaded autobackup_5.12.35_20191216_1825_1576520700002.unf in 382ms
    Dec 16 19:25:07 hostname unifibackup[7791]: uploaded autobackup_5.12.35_20191216_1925_1576524300011.unf in 368ms

## IAM Policy

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

*N.B. if using EC2, an [instance profile](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use_switch-role-ec2.html) can make management much easier.*
