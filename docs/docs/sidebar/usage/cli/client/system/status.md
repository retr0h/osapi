# Status

Get the system status:

```bash
$ osapi client system status
11:03AM INF client configuration debug=false client.url=http://0.0.0.0:8080

  Hostname: nerd
  Uptime: 0 days, 23 hours, 1 minute
  Load Average (1m, 5m, 15m): 2.27, 2.10, 2.00
  Memory: 18 GB used / 31 GB total / 0 GB free

  Disks:

  ┏━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━┓
  ┃     DISK NAME      ┃       TOTAL        ┃        USED        ┃        FREE        ┃
  ┣━━━━━━━━━━━━━━━━━━━━╋━━━━━━━━━━━━━━━━━━━━╋━━━━━━━━━━━━━━━━━━━━╋━━━━━━━━━━━━━━━━━━━━┫
  ┃ /                  ┃ 97 GB              ┃ 56 GB              ┃ 36 GB              ┃
  ┃ /boot              ┃ 1 GB               ┃ 0 GB               ┃ 1 GB               ┃
  ┃ /var/lib/kubelet/p ┃ 97 GB              ┃ 56 GB              ┃ 36 GB              ┃
  ┃ ods/a646b44c-19bf- ┃                    ┃                    ┃                    ┃
  ┃ 421e-8825-         ┃                    ┃                    ┃                    ┃
  ┃ 53c1d1aa124b/volum ┃                    ┃                    ┃                    ┃
  ┃ e-subpaths/custom- ┃                    ┃                    ┃                    ┃
  ┃ dnsmasq/pihole/1   ┃                    ┃                    ┃                    ┃
  ┃ /var/lib/kubelet/p ┃ 97 GB              ┃ 56 GB              ┃ 36 GB              ┃
  ┃ ods/a646b44c-19bf- ┃                    ┃                    ┃                    ┃
  ┃ 421e-8825-         ┃                    ┃                    ┃                    ┃
  ┃ 53c1d1aa124b/volum ┃                    ┃                    ┃                    ┃
  ┃ e-subpaths/custom- ┃                    ┃                    ┃                    ┃
  ┃ dnsmasq/pihole/2   ┃                    ┃                    ┃                    ┃
  ┃ /var/lib/kubelet/p ┃ 97 GB              ┃ 56 GB              ┃ 36 GB              ┃
  ┃ ods/1bb4bf24-ecf5- ┃                    ┃                    ┃                    ┃
  ┃ 4194-adbb-         ┃                    ┃                    ┃                    ┃
  ┃ 5d8697325795/volum ┃                    ┃                    ┃                    ┃
  ┃ e-                 ┃                    ┃                    ┃                    ┃
  ┃ subpaths/config/gr ┃                    ┃                    ┃                    ┃
  ┃ afana/0            ┃                    ┃                    ┃                    ┃
  ┃ /var/lib/kubelet/p ┃ 97 GB              ┃ 56 GB              ┃ 36 GB              ┃
  ┃ ods/1bb4bf24-ecf5- ┃                    ┃                    ┃                    ┃
  ┃ 4194-adbb-         ┃                    ┃                    ┃                    ┃
  ┃ 5d8697325795/volum ┃                    ┃                    ┃                    ┃
  ┃ e-subpaths/sc-     ┃                    ┃                    ┃                    ┃
  ┃ dashboard-         ┃                    ┃                    ┃                    ┃
  ┃ provider/grafana/3 ┃                    ┃                    ┃                    ┃
  ┗━━━━━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━━━━━┛
```
