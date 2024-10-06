# Delete

Delete a action from the task runner by `message-id`:

```bash
$ osapi client task delete --message-id 5
10:17AM INF task delete message_id=5 status=ok
```

## Errors

### Not Found

```bash
$ osapi client task delete --message-id 5
10:17AM ERR failed to get message from the task error="no item found with ID: 5"
exit status 1
```
