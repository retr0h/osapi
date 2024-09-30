# Put

Add an item to the queue by `message-body`:

:::note

The `message-body` must be a Base64 encoded binary Protobuf message.

:::

```bash
$ osapi client queue put --message-body 'EhIKBzguOC44LjgKBzguOC40LjQ='
5:34PM INF queue put message_body='EhIKBzguOC44LjgKBzguOC40LjQ=' response="" status=ok
```
