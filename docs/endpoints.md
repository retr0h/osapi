## **RESTful API Endpoints for Linux System Configuration**

### **1. System Information and Management**
- **Information**
  - `/system/info` - Get system information (hostname, OS version, uptime, etc.).
  - `/system/logs` - Access system logs.
  - `/system/status` - Get overall system status (load, memory usage, etc.).
- **Time and Localization**
  - `/system/time/zone` - Manage time zone.
  - `/system/time/ntp` - Manage NTP settings.
  - `/system/time/sync` - Synchronize time with NTP servers.
  - `/system/localization/language` - Manage system language settings.
  - `/system/localization/locale` - Manage system locale settings.
- **Power Management**
  - `/system/power/shutdown` - Shutdown the system.
  - `/system/power/reboot` - Reboot the system.
  - `/system/power/hibernate` - Hibernate the system.

### **2. User and Group Management**
- `/users` - Manage users.
- `/users/{username}` - Manage a specific user.
- `/groups` - Manage groups.
- `/groups/{groupname}` - Manage a specific group.

### **3. Package Management**
- `/packages` - List installed packages.
- `/packages/install` - Install a package.
- `/packages/remove` - Remove a package.
- `/packages/update` - Update a package or all packages.

### **4. Networking**
- **Interfaces**
  - `/network/interfaces`
  - `/network/interfaces/{interface-name}`
- **DNS**
  - `/network/dns`
  - `/network/dns/{dns-server}`
- **Routes**
  - `/network/routes`
  - `/network/routes/{route-id}`
- **Firewall**
  - `/network/firewall/rules`
  - `/network/firewall/rules/{rule-id}`
- **Hostname**
  - `/network/hostname`

### **5. Storage Management**
- **Disks**
  - `/storage/disks` - List disks.
  - `/storage/disks/{disk-id}` - Manage a specific disk.
- **Partitions**
  - `/storage/partitions` - List partitions.
  - `/storage/partitions/{partition-id}` - Manage a specific partition.
- **Mounts**
  - `/storage/mounts` - List mounted filesystems.
  - `/storage/mounts/{mount-id}` - Manage a specific mount.
- **File Systems**
  - `/storage/filesystems` - List file systems.
  - `/storage/filesystems/{filesystem-id}` - Manage a specific file system.

### **6. Process Management**
- `/processes` - List running processes.
- `/processes/{pid}` - Manage a specific process (e.g., kill, nice).

### **7. Services and Daemons**
- `/services` - List running services.
- `/services/{service-name}` - Manage a specific service (start, stop, restart, enable, disable).

### **8. NTP (Network Time Protocol)**
- `/services/ntp/servers`
- `/services/ntp/servers/{server-id}`
- `/services/ntp/config`
- `/services/ntp/peers`
- `/services/ntp/peers/{peer-id}`

### **9. Cron Jobs**
- `/services/cron/jobs`
- `/services/cron/jobs/{job-id}`

### **10. Security**
- **SELinux/AppArmor**
  - `/security/selinux/status` - Get SELinux status.
  - `/security/apparmor/status` - Get AppArmor status.
- **Users and Groups**
  - `/security/users` - List users with security roles.
  - `/security/groups` - List groups with security roles.
- **SSH**
  - `/security/ssh/config` - Manage SSH configuration.
  - `/security/ssh/keys` - Manage SSH keys.

### **11. Backup and Restore**
- `/backup` - Perform or schedule system backups.
- `/restore` - Restore from a backup.

### **12. Monitoring and Alerts**
- `/monitoring/metrics` - Access system metrics.
- `/monitoring/alerts` - Manage monitoring alerts and notifications.
