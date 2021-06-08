# proxy_checker

Приложение, для проверки статуса прокси сервера, с выбранным диапазоном портов. 

## Motivation

Возникла необходимость проверить прокси-сервер и "простучать" с **9000** порта по **10000**, также сделать это асинхронно, с указанием таймаута. После проверки записать ответы в виде отчётов (успешные, ошибочные, унакальность ip). 

## Check ip (URLs)

В качестве проверки на уникальность IP, можно использовать один из URL:

- https://ip.nf/me.json
- https://checker.soax.com/api/ipinfo

Если будет указан другой URL, проверка на IP произведена не будет.

## Labels

- `%PORT%` - Используется, для подстановки проверяемого порта в URL `-proxy-host`.

## Build

Чтобы собрать приложение, необходимо выполнить:

```
make build_darwin_amd64
```

## Run:

```bash
./proxy_checker.darwin-amd64 \
-proxy-host="http://userpane:pass@example.com:%PORT%" \
-dest="https://ip.nf/me.json" \
-async=100 \
-timeout=60 \
-proxy-port-from=9000 \
-proxy-port-to=9999 \
-reports=true
```

Or

```bash
./proxy_checker.darwin-amd64 \
-proxy-host="http://userpane-port-%PORT%:pass@example.com:1234" \
-dest="https://ip.nf/me.json" \
-async=100 \
-timeout=60 \
-proxy-port-from=9000 \
-proxy-port-to=9999 \
-reports=true
```

## TODO

- [ ] - Добавить для сборки https://github.com/goreleaser/goreleaser
- [ ] - Добавить сортировку по кол-ву IP в отчёте
- [ ] - Добавить проверку IP на любые URL

