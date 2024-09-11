# Delete

```bash
$ osapi queue client delete --message-id m_f222bd10b207fdbbfb83ec88ae5f17bb
10:17AM INF queue configuration debug=false queue.database.driver_name=sqlite3 queue.database.data_source_name="file:database.db?_journal=WAL&_timeout=5000&_fk=true" queue.database.max_open_conns=1 queue.database.max_idle_conns=1
10:17AM INF queue delete messageID=m_f222bd10b207fdbbfb83ec88ae5f17bb status=ok
```

## Errors

### Not Found

```bash
$ osapi queue client delete --message-id m_f222bd10b207fdbbfb83ec88ae5f17bb
10:17AM INF queue configuration debug=false queue.database.driver_name=sqlite3 queue.database.data_source_name="file:database.db?_journal=WAL&_timeout=5000&_fk=true" queue.database.max_open_conns=1 queue.database.max_idle_conns=1
10:17AM ERR failed to get message from the queue error="no item found with ID: m_f222bd10b207fdbbfb83ec88ae5f17bb"
exit status 1
```
