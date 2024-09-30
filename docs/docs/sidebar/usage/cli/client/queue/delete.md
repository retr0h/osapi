# Delete

Delete an item from the queue by `message-id`:

```bash
$ osapi client queue delete --message-id m_f222bd10b207fdbbfb83ec88ae5f17bb
10:17AM INF queue delete messageID=m_f222bd10b207fdbbfb83ec88ae5f17bb status=ok
```

## Errors

### Not Found

```bash
$ osapi client queue delete --message-id m_f222bd10b207fdbbfb83ec88ae5f17bb
10:17AM ERR failed to get message from the queue error="no item found with ID: m_f222bd10b207fdbbfb83ec88ae5f17bb"
exit status 1
```
