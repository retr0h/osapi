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

5:07PM INF client configuration debug=false client.url=http://0.0.0.0:8080
5:07PM INF response code=200 hostname=c6072f8815fe uptime="1 day, 0 hours, 18 minutes" load.1m=0 load.5m=0 load.15m=0 memory.total=16788750336 memory.free=12436840448 memory.used=1214160896
```
