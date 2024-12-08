# Update

Update the systems DNS config:

```bash
$ osapi client network dns get update --search-domains "foo,bar,baz" --servers "1.1.1.1,2.2.2.2" --interface-name eth1
10:56AM INF network dns put search_domains=foo,bar,baz servers=1.1.1.1,2.2.2.2 response="" status=ok
```
