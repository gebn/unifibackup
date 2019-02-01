# UniFi Backup

[![Build Status](https://travis-ci.org/gebn/unifibackup.svg?branch=master)](https://travis-ci.org/gebn/unifibackup)
[![GoDoc](https://godoc.org/github.com/gebn/unifibackup?status.svg)](https://godoc.org/github.com/gebn/unifibackup)
[![Go Report Card](https://goreportcard.com/badge/github.com/gebn/unifibackup)](https://goreportcard.com/report/github.com/gebn/unifibackup)

A daemon that monitors a UniFi Controller's `autobackup` directory, uploading new backups to S3 as soon as they are ready.

## Controller Setup

Enable auto backup under `Settings > Auto Backup` on your controller. Set the occurrence to as often as your internet connection can take - the more frequent, the less data you are likely to lose. You only need to retain one (i.e. the latest) backup, however you may want to keep around more; this daemon will work correctly regardless, as it only pays attention to new files.

## Systemd Setup

The executable is intended to run under systemd. The following instructions detail how to set this up

1. Open up `unifibackup.service` in your favourite editor.
   1. Change the bucket to the one you want to upload to (see *IAM Policy* below for required permissions), and optionally override the destination prefix.
   2. Set `AWS_REGION` to the region of the bucket above.
   3. If not using an instance profile, and credentials are not configured elsewhere, set the `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables.

2. Execute the following as `root`:

       cp unifibackup.service /etc/systemd/system
       systemctl daemon-reload                     # pick up changes
       systemctl enable unifibackup.service        # start on boot
       systemctl start unifibackup.service         # start right now
       systemctl status unifibackup.service        # check running smoothly (look for "active (running)")

Running the final `status` command again after a few multiples of the backup frequency should show the daemon successfully uploading new files:

    $ sudo systemctl status unifibackup.service
    ● unifibackup.service - A utility to upload Unifi Controller backups to S3
       Loaded: loaded (/etc/systemd/system/unifibackup.service; enabled; vendor preset: enabled)
       Active: active (running) since Sun 2018-10-21 22:50:03 UTC; 20h ago
         Docs: https://github.com/gebn/unifibackup/blob/master/README.md
     Main PID: 13790 (unifibackup)
        Tasks: 10 (limit: 4915)
       CGroup: /system.slice/unifibackup.service
               └─13790 /usr/local/bin/unifibackup -bucket bucket.example.com

    Oct 22 10:20:05 hostname unifibackup[13790]: Uploaded autobackup_5.9.29_20181022_1020_1540203600005.unf in 551ms
    Oct 22 11:20:05 hostname unifibackup[13790]: Uploaded autobackup_5.9.29_20181022_1120_1540207200006.unf in 807ms
    Oct 22 12:20:05 hostname unifibackup[13790]: Uploaded autobackup_5.9.29_20181022_1220_1540210800007.unf in 559ms
    Oct 22 13:20:05 hostname unifibackup[13790]: Uploaded autobackup_5.9.29_20181022_1320_1540214400006.unf in 701ms
    Oct 22 14:20:05 hostname unifibackup[13790]: Uploaded autobackup_5.9.29_20181022_1420_1540218000007.unf in 732ms
    Oct 22 15:20:05 hostname unifibackup[13790]: Uploaded autobackup_5.9.29_20181022_1520_1540221600006.unf in 639ms
    Oct 22 16:20:05 hostname unifibackup[13790]: Uploaded autobackup_5.9.29_20181022_1620_1540225200005.unf in 495ms
    Oct 22 17:20:06 hostname unifibackup[13790]: Uploaded autobackup_5.9.29_20181022_1720_1540228800007.unf in 710ms
    Oct 22 18:20:06 hostname unifibackup[13790]: Uploaded autobackup_5.9.29_20181022_1820_1540232400008.unf in 616ms
    Oct 22 19:20:06 hostname unifibackup[13790]: Uploaded autobackup_5.9.29_20181022_1920_1540236000007.unf in 549ms

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
