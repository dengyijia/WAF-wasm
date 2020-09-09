# Performance Tests

We have conducted some performance tests for the WAF WASM filter using `Fortio`. Specifically, we deployed the filter on the product page of Istio sample application Bookinfo, and measured the latency of safe requests at 50th, 75th, 90th, 99th, and 99.9th percentile. The baseline for comparison is the latency for Bookinfo application without the filter. To see the results, jump to the [Results](README.md#Results) section below.

In the performance tests, we set the default number of queries per second(QPS) to be 1000 and the default number of client connections to be 16. We made measurements for QPS = 10, 50, 100, 500, 1000, 2000 and number of connections = 2, 4, 8, 16, 32, 64 both with and without the jitter option in `Fortio`. Thus, there are 24 tests in a set of measurements.

The procedure for performance tests is as follows:
1. Run Bookinfo application without the WAF WASM filter and make measurements
2. Run Bookinfo application with the WAF WASM filter and make measurements
3. Analyze and plot the results

Step 1 and 2 can be done using the [`performance_test.sh`](./performance_test.sh) script. The instructions for running Bookinfo without the filter can be found in Istio documentation [here](https://istio.io/latest/docs/examples/bookinfo/), and the instructions for deploying the filter are in [DEPLOY.md](../DEPLOY.md) in this repository. After Bookinfo has been running and the ingress gateway has been exported, we used the following command to make measurements:
```
./performance_test.sh <status> <gateway_url>
```
where `<status>` is either `deployed` or `undeployed` and `<gateway_url>` is the ingress gateway URL for Bookinfo.
Since there are 24 tests in total and each test takes 4 minutes, the test will run for quite a while. After the tests complete, the results will be in the [data](data) folder as `json` files.

After measurements for both deployed and undeployed cases are measured, we used [`plot.py`](plot.py) to analyze the results.
```
python3 plot.py
```
## Results

