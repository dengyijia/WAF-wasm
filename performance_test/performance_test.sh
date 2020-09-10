#!/bin/bash

# STATUS: either "deployed" or "undeployed"
STATUS="$1"

# GATEWAY_URL: ingress gateway url to bookinfo
GATEWAY_URL="$2"

LABEL_PREFIX="WAF_wasm_${STATUS}"

DIR="json"
TIME==$(date "+%Y.%m.%d-%H.%M.%S")
NEW_DIR="${LABEL_PREFIX}_${TIME}"
if [ "$#" -ne 2 ]; then
  NEW_DIR="$3"

DEFAULT_QPS=1000
DEFAULT_CONN=16

JITTERS=(True False)
CONN=(2 4 8 16 32 64)
QPS=(10 50 100 500 1000 2000)

cd ${DIR}
mkdir ${NEW_DIR}
cd ${NEW_DIR}
for jitter in ${JITTERS[@]}; do

  # test for varying number of connections
  for conn in ${CONN[@]}; do
    LABEL="${LABEL_PREFIX}_jitter=${jitter}_conn=${conn}_qps=${DEFAULT_QPS}"
    echo "Performance Test for $LABEL"
    fortio load -jitter=${jitter} -c ${conn} -qps ${DEFAULT_QPS} -n 15000 -a -r 0.001 -httpbufferkb=128 -labels "${LABEL}"  http://${GATEWAY_URL}/productpage
  done

  # test for varying number of queries per second
  for qps in ${QPS[@]}; do
    LABEL="${LABEL_PREFIX}_jitter=${jitter}_conn=${DEFAULT_CONN}_qps=${qps}"
    echo "Performance Test for $LABEL"
    fortio load -jitter=${jitter} -c ${DEFAULT_CONN} -qps ${qps} -n 15000 -a -r 0.001 -httpbufferkb=128 -labels "${LABEL}"  http://${GATEWAY_URL}/productpage
  done
done
cd ../..
