# List

List item from the queue by page and offset:

```bash
$ osapi client queue list

  Total Items: 1
  Total Pages: 1

  Items:

  ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
  ┃                        ID                        ┃                     CREATED                      ┃                       TASK                       ┃
  ┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╋━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╋━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
  ┃ m_cdb5ca64cc494646eee5edc7f8bc3313               ┃ 2024-09-29T20:28:14Z                             ┃ change_dns_action:  {                            ┃
  ┃                                                  ┃                                                  ┃   dns_servers:  "8.8.8.8"                        ┃
  ┃                                                  ┃                                                  ┃   dns_servers:  "8.8.4.4"                        ┃
  ┃                                                  ┃                                                  ┃ }                                                ┃
  ┃                                                  ┃                                                  ┃                                                  ┃
  ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```