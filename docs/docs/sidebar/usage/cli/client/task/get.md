# Get

Get a task action by `message-id`:

```bash
$ osapi client task get --message-id 5


  ID: 5
  Created: 2024-09-29T20:28:14Z

  ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
  ┃                                               ACTION                                               ┃
  ┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
  ┃ change_dns_action:  {                                                                              ┃
  ┃   dns_servers:  "8.8.8.8"                                                                          ┃
  ┃   dns_servers:  "8.8.4.4"                                                                          ┃
  ┃ }                                                                                                  ┃
  ┃                                                                                                    ┃
  ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```

## Errors

### Not Found

```bash
$ osapi client task get get --message-id 10000
7:03PM ERR not found code=404 response="not found: no item found with ID 10000"
```
