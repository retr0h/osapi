---
sidebar_position: 4
---

# Usage

## Server

Start the API server:

```bash
osapi server start
```

## CLI

### Ping

Ping the api:

```bash
osapi client ping

6:12PM INF client configuration debug=false client.url=http://0.0.0.0:8080
6:12PM INF response code=200 data=pong
```

### System

#### Status

Get the system status:

```bash
osapi client system status

9:47PM INF client configuration debug=false client.url=http://0.0.0.0:8080
9:47PM INF response code=200 hostname=55372261bbbe uptime="2 days, 4 hours, 58 minutes" load.1m=0 load.5m=0 load.15m=0 memory.Total=16788750336 memory.Free=686661632 memory.Used=14723842048 disks.0.Name=/etc/resolv.conf disks.0.Total=270233210880 disks.0.Used=20075712512 disks.0.Free=236396883968 disks.1.Name=/etc/hostname disks.1.Total=270233210880 disks.1.Used=20075712512 disks.1.Free=236396883968 disks.2.Name=/etc/hosts disks.2.Total=270233210880 disks.2.Used=20075712512 disks.2.Free=236396883968
```
