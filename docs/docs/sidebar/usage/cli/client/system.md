# System

## Status

:::note

The following disk layout took place on a Docker container.

:::

Get the system status:

```bash
$ osapi client system status
9:30PM INF client configuration debug=false client.url=http://0.0.0.0:8080

  Hostname: ca946721bd50
  Uptime: 3 days, 4 hours, 41 minutes
  Load Average (1m, 5m, 15m): 0.00, 0.00, 0.07
  Memory: 13 GB used / 15 GB total / 0 GB free

  Disks:

  ┏━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━┓
  ┃     DISK NAME      ┃       TOTAL        ┃        USED        ┃        FREE        ┃
  ┣━━━━━━━━━━━━━━━━━━━━╋━━━━━━━━━━━━━━━━━━━━╋━━━━━━━━━━━━━━━━━━━━╋━━━━━━━━━━━━━━━━━━━━┫
  ┃ /etc/resolv.conf   ┃ 251 GB             ┃ 19 GB              ┃ 219 GB             ┃
  ┃ /etc/hostname      ┃ 251 GB             ┃ 19 GB              ┃ 219 GB             ┃
  ┃ /etc/hosts         ┃ 251 GB             ┃ 19 GB              ┃ 219 GB             ┃
  ┗━━━━━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━━━━━┛
```
