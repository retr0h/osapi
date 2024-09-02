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

10:38AM INF client configuration debug=false client.url=http://0.0.0.0:8080
10:38AM INF response code=200 hostname=6b714859c1f0 uptime="0 days, 0 hours, 15 minutes" load.1m=0 load.5m=0 load.15m=0
```
