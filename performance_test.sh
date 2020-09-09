#!/bin/bash
GATEWAY_URL="$1"
LABEL_PREFIX="WAF_wasm"

JITTERS=(True False)
CONNECTIONS=(2 4 8 16 32 64)
QPS=(10 50 100 500 1000 2000)

for jitter in ${JITTERS[@]}; do
  for connection in ${CONNECTIONS[@]}; do
    for qps in ${QPS[@]}; do
      LABEL="${LABEL_PREFIX}_jitter=${jitter}_connection=${connection}_qps=${qps}"
      echo $LABEL
      fortio load  -jitter=${jitter} -c ${connection} -qps ${qps} -t 240s -a -r 0.001 -httpbufferkb=128 -labels "${LABEL}"  http://${GATEWAY_URL}/productpage
    done
  done
done

