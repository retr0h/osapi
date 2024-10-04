---
sidebar_position: 6
---

# API Design Guidelines

1. **General System Information or Status**

- **Path**: `/system/`
- **Description**: Endpoints that represent broader operations impacting the
  entire system, such as system status or information, should stay under
  `/system/`.
- **Examples**:
  - `/system/status`
  - `/system/info`

2. **Functional Areas**

Functional areas like power management, NTP, or network management should be
broken into separate categories if they involve multiple endpoints or represent
distinct areas of configuration.

- Examples:
  - **Power Management**:
    - `/power/` - Power-related operations (shutdown, reboot, hibernate).
  - **NTP Management**:
    - `/ntp/` - NTP settings management.
    - `/ntp/sync` - Time synchronization with NTP servers.
  - **Network Management**:
    - `/network/` - Network interfaces, routes, and DNS management.

3. **Consider Scalability and Future Needs**

If an area is expected to grow in complexity with more endpoints, it's best to
separate it early into its own category, even if it only has a few operations
today. This ensures future scalability and avoids clutter in the core `/system/`
path.
