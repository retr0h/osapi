# Add

Add an action to the task runner by `message-body`:

:::warning

This sub-command will likely be removed.

:::

```bash
$ osapi client task add --proto-file examples/dns.bin



  ID: 43
```

## Errors

### Not Found

```bash
$ osapi client task get add --proto-file examples/dns-invalid.bin
7:06PM ERR bad request code=400 response="Body field is required and cannot be empty"
```
