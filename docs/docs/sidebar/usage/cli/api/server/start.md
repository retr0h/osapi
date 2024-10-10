# Start

To start the server:

```bash
$ osapi server start
2:24AM INF server configuration debug=false server.port=8080 server.security.cors.allow_origins="[http://localhost:3001 https://retr0h.github.io]"
⇨ http server started on [::]:8080
```

## Least Privilege Mode

We aim to run this API service with the **least privilege mode** to maximize
security while reading data from Linux. This means that the API will only return
data that the running user has permission to access. The API is designed to
gracefully skip over partitions or other system resources where permission
errors occur (e.g., due to lack of root access).

If your goal is to run the API with minimal limitations, you will need to run
the API daemon as `root`.

However, when run as a non-root user, the service will:

- Collect and return available disk usage statistics for partitions it has
  permission to access
- Skip partitions or system paths that result in "permission denied" errors
- Attempt to send an "unprivileged" ping via UDP

Running as a regular user maintains a secure, restricted mode, but some
functionality (such as access to certain system directories and files) will be
limited.

If full access to system resources is required (e.g., to access all disk
partitions or perform privileged operations), running the API daemon as `root`
is necessary.

### ICMP Permissions

The API can still send pings without requiring root access, but on Linux, this
requires modifying system settings using the following sysctl command:

```bash
$ sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"
```

Alternatively, running the API as root will allow full access to privileged
operations like raw socket pinging.
