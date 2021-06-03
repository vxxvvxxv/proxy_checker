#!/usr/bin/env bash
# Vars
DEST_ADDRESS="https://ip.nf/me.json"
ASYNC_REQUESTS=100
TIMEOUT_TIME=60
LABEL_PORT="%PORT%"
# -----------------
PROXY_USER="USER-SESSION-ID-$LABEL_PORT"
PROXY_PASSWORD="UniqPassword"
PROXY_ADDRESS="proxy.example.com:1234"
PROXY_URL="http://$PROXY_USER:$PROXY_PASSWORD@$PROXY_ADDRESS"
# -----------------

# Run tests
./bin/proxy_checker -proxy-host=$PROXY_URL -proxy-port-from=1000 -proxy-port-to=2000 -dest=$DEST_ADDRESS -async=$ASYNC_REQUESTS -timeout=$TIMEOUT_TIME
