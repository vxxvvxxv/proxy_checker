# proxy_checker

Proxy checker with send async requests and save report (success, error, ip)

TODO: Add sort by count IP

## Check ip (URLs)

- https://ip.nf/me.json
- https://checker.soax.com/api/ipinf

## Labels

- `%PORT%` - using for selected port.

## Run:

```bash
go run -race ./main.go -reports=true -proxy-host="http://userpane:pass@example.com:%PORT%" -dest="https://ip.nf/me.json" -async=100 -timeout=60 -proxy-port-from=9000 -proxy-port-to=9999 -reports=true
```

Or

```bash
go run -race ./main.go -reports=true -proxy-host="http://userpane-port-%PORT%:pass@example.com:1234" -dest="https://ip.nf/me.json" -async=100 -timeout=60 -proxy-port-from=9000 -proxy-port-to=9999 -reports=true
```
