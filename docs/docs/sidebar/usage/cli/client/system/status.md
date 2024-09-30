# Status

Get the system status:

```bash
$ osapi client system status

  Hostname: nerd
  Uptime: 12 days, 5 hours, 32 minutes
  Load Average (1m, 5m, 15m): 1.56, 1.91, 2.08
  Memory: 19 GB used / 31 GB total / 0 GB free

  Disks:

  ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
  ┃              DISK NAME               ┃                TOTAL                 ┃                 USED                 ┃                 FREE                 ┃
  ┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╋━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╋━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╋━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
  ┃ /                                    ┃ 97 GB                                ┃ 56 GB                                ┃ 35 GB                                ┃
  ┃ /boot                                ┃ 1 GB                                 ┃ 0 GB                                 ┃ 1 GB                                 ┃
  ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```