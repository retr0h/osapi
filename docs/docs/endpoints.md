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

- `/ntp/servers`
- `/ntp/servers/{server-id}`
- `/ntp/config`
- `/ntp/peers`
- `/ntp/peers/{peer-id}`

### Cron Jobs

- `/cron/jobs`
- `/cron/jobs/{job-id}`

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

Core System Management: /system/info - Get system information (hostname, OS
version, uptime, etc.). /system/status - Get overall system status (load, memory
usage, etc.). /system/logs - Access system logs. Time Management (Standalone
Category): /ntp - Manage NTP settings. /ntp/sync - Synchronize time with NTP
servers. /time/zone - Manage time zone (or include in /ntp/ if related). Power
Management (Standalone Category): /power/shutdown - Shutdown the system.
/power/reboot - Reboot the system. /power/hibernate - Hibernate the system.
Localization (Keep Nested Under System): /system/localization/language - Manage
system language settings. /system/localization/locale - Manage system locale
settings. By breaking out NTP and power into their own top-level categories, you
keep the structure modular, scalable, and easy to understand, while still
nesting system-specific operations under /system/.
