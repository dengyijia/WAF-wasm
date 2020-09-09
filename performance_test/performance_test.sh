#!/bin/bash
STATUS="$1"
GATEWAY_URL="$2"
LABEL_PREFIX="WAF_wasm_${STATUS}"
DIR="data"

DEFAULT_QPS=1000
DEFAULT_CONN=16

JITTERS=(True False)
CONN=(2 4 8 16 32 64)
QPS=(10 50 100 500 1000 2000)

cd ${DIR}
for jitter in ${JITTERS[@]}; do

  # test for varying number of connections
  for conn in ${CONN[@]}; do
    LABEL="${LABEL_PREFIX}_jitter=${jitter}_conn=${conn}_qps=${DEFAULT_QPS}"
    echo "Performance Test for $LABEL"
    fortio load -jitter=${jitter} -c ${conn} -qps ${DEFAULT_QPS} -t 240s -a -r 0.001 -httpbufferkb=128 -labels "${LABEL}"  http://${GATEWAY_URL}/productpage
  done

  # test for varying number of queries per second
  for qps in ${QPS[@]}; do
    LABEL="${LABEL_PREFIX}_jitter=${jitter}_conn=${DEFAULT_CONN}_qps=${qps}"
    echo "Performance Test for $LABEL"
    fortio load -jitter=${jitter} -c ${DEFAULT_CONN} -qps ${qps} -t 240s -a -r 0.001 -httpbufferkb=128 -labels "${LABEL}"  http://${GATEWAY_URL}/productpage
  done
done
cd ..
