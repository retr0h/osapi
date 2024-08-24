---
sidebar_position: 2
---

# Endpoints

The client and server components are generated from an OpenAPI spec.

## RESTful API Endpoints for Linux System Configuration

### User and Group Management

- `/users` - Manage users.
- `/users/{username}` - Manage a specific user.
- `/groups` - Manage groups.
- `/groups/{groupname}` - Manage a specific group.

### Package Management

- `/packages` - List installed packages.
- `/packages/install` - Install a package.
- `/packages/remove` - Remove a package.
- `/packages/update` - Update a package or all packages.

### Networking

- Interfaces
  - `/network/interfaces`
  - `/network/interfaces/{interface-name}`
- DNS
  - `/network/dns`
  - `/network/dns/{dns-server}`
- Routes
  - `/network/routes`
  - `/network/routes/{route-id}`
- Firewall
  - `/network/firewall/rules`
  - `/network/firewall/rules/{rule-id}`
- Hostname
  - `/network/hostname`

### Storage Management

- Disks
  - `/storage/disks` - List disks.
  - `/storage/disks/{disk-id}` - Manage a specific disk.
- Partitions
  - `/storage/partitions` - List partitions.
  - `/storage/partitions/{partition-id}` - Manage a specific partition.
- Mounts
  - `/storage/mounts` - List mounted filesystems.
  - `/storage/mounts/{mount-id}` - Manage a specific mount.
- File Systems
  - `/storage/filesystems` - List file systems.
  - `/storage/filesystems/{filesystem-id}` - Manage a specific file system.

### Process Management

- `/processes` - List running processes.
- `/processes/{pid}` - Manage a specific process (e.g., kill, nice).

### Services and Daemons

- `/services` - List running services.
- `/services/{service-name}` - Manage a specific service (start, stop, restart,
  enable, disable).

### NTP (Network Time Protocol)

- `/services/ntp/servers`
- `/services/ntp/servers/{server-id}`
- `/services/ntp/config`
- `/services/ntp/peers`
- `/services/ntp/peers/{peer-id}`

### Cron Jobs

- `/services/cron/jobs`
- `/services/cron/jobs/{job-id}`

### Security

- SELinux/AppArmor
  - `/security/selinux/status` - Get SELinux status.
  - `/security/apparmor/status` - Get AppArmor status.
- Users and Groups
  - `/security/users` - List users with security roles.
  - `/security/groups` - List groups with security roles.
- SSH
  - `/security/ssh/config` - Manage SSH configuration.
  - `/security/ssh/keys` - Manage SSH keys.

### Backup and Restore

- `/backup` - Perform or schedule system backups.
- `/restore` - Restore from a backup.

### Monitoring and Alerts

- `/monitoring/metrics` - Access system metrics.
- `/monitoring/alerts` - Manage monitoring alerts and notifications.
