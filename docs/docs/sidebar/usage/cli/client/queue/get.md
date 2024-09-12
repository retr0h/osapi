# Get

```bash
$ osapi client queue get --message-id m_b9f89ed63c04264776695e30b085c1dc
9:11PM INF queue configuration debug=false queue.database.driver_name=sqlite3 queue.database.data_source_name="file:database.db?_journal=WAL&_timeout=5000&_fk=true" queue.database.max_open_conns=1 queue.database.max_idle_conns=1

  ID: m_b9f89ed63c04264776695e30b085c1dc
  Created: 2024-09-11T04:11:05Z
  Updated: 2024-09-11T04:11:05Z
  Timeout: 2024-09-10T21:11:05-07:00
  Received: 0

  ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
  ┃                       Body                       ┃
  ┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
  ┃ yo                                               ┃
  ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```
