# proxy_checker

Proxy checker with send async requests and save report (success, error, ip)

TODO: Add sort by count IP

## Run:

```bash
go run -race ./main.go -reports=true -proxy-host="http://userpane:pass@example.com" -dest="https://checker.soax.com/api/ipinfo" -async=100 -timeout=60 -proxy-port-from=9000 -proxy-port-to=9999
```
