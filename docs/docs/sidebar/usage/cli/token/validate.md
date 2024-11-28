# Validate

Validate a JSON Web Token (JWT) by checking its signature, expiration, and claims:

```bash
$ osapi token validate --secret-key foo --token eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlcyI6WyJyZWFkIl0sImlzcyI6Im9zYXBpIiwic3ViIjoidXNlcjEyMyIsImV4cCI6MTc0MDg0Nzg4NywiaWF0IjoxNzMyODk5MDg3fQ.c_gYUOFzeg7GOraAtTRuyH_g_u4KluuuGATITVdQu2E


  Roles: read
  Subject: user123
  Audience:
  Expires: 2025-03-01T08:51:27-08:00
  Issued: 2024-11-29T08:51:27-08:00
```
