#!/bin/bash
set -e
ETC_HOSTS_PROXY_BIND_ADDRESS="${ETC_HOSTS_PROXY_BIND_ADDRESS:-0.0.0.0:8080}"
ETC_HOSTS_PROXY_MODE="${ETC_HOSTS_PROXY_MODE:-http}"
ADDITIONAL_ARGS=""

if [ ! -z "$ETC_HOSTS_PROXY_BIND_ADDRESS" ]; then
  ADDITIONAL_ARGS="$ADDITIONAL_ARGS -L $ETC_HOSTS_PROXY_BIND_ADDRESS"
fi

if [ ! -z "$ETC_HOSTS_PROXY_MODE" ]; then
  ADDITIONAL_ARGS="$ADDITIONAL_ARGS -M $ETC_HOSTS_PROXY_MODE"
fi

if [ ! -z "$ETC_HOSTS_PROXY_HOSTS_LIST" ]; then
  ADDITIONAL_ARGS="$ADDITIONAL_ARGS -H $ETC_HOSTS_PROXY_HOSTS_LIST"
fi

if [[ "$1" == "run" ]] && [[ $# -eq 1 ]]; then
  if [[ -z "$ETC_HOSTS_PROXY_HOSTS_LIST" ]] ; then
    echo "Add some hosts to the list or use full args in docker run: $0 run -M http -L 0.0.0.0:8080 -H <your_host>=<your_ip>"
    exit -1
  fi
  echo "Start proxy with args: $0 run $ADDITIONAL_ARGS"
  /usr/bin/etc-hosts-proxy "$* $ADDITIONAL_ARGS"
else
  echo "Start proxy with args: $0 $*"
  /usr/bin/etc-hosts-proxy "$@"
fi
