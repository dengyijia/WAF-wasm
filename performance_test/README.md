# Performance Tests

We have conducted some performance tests for the WAF WASM filter using [`Fortio`](https://github.com/fortio/fortio). Specifically, we deployed the filter on the product page of Istio sample application Bookinfo, and measured the latency of safe requests at 50th, 75th, 90th, 99th, and 99.9th percentile. The baseline for comparison is the latency of Bookinfo application without the filter. To see the results, go to the [csv](csv) and [figs](figs) folder.

In the performance tests, we set the default number of queries per second(QPS) to be 1000 and the default number of client connections to be 16. We made measurements for QPS = 10, 50, 100, 500, 1000, 2000 and number of connections = 2, 4, 8, 16, 32, 64 both with and without the jitter option in `Fortio`. Thus, there are 24 sets of parameters to measure. For each set, we made 15000 requests.

The procedure for performance tests is as follows:
1. Run Bookinfo application without the WAF WASM filter and make measurements
2. Run Bookinfo application with the WAF WASM filter and make measurements
3. Process results and plot the results

### Measurements
Step 1 and 2 can be done using the [`performance_test.sh`](./performance_test.sh) script. The instructions for running Bookinfo can be found in Istio documentation [here](https://istio.io/latest/docs/examples/bookinfo/), and the instructions for deploying the filter are in [DEPLOY.md](../DEPLOY.md). After Bookinfo has been running and the ingress gateway has been exported, we can use the following command to make measurements:
```
./performance_test.sh <status> <gateway_url> <label>
```
where `<status>` is either `deployed` or `undeployed`, `<gateway_url>` is the ingress gateway URL for Bookinfo, and `<label>` is any label you want to give for the results. `<label>` is optional; if it is not provided, it will default to `WAF_wasm_test_result`. The label for the deployed and the undeployed case should be the same.

Since there are 24 sets of parameters to test and each set makes 15000 requests, the test will run for quite a while. After the tests complete, the results will be in the `json/<label>` folder.

### Results and Plots
After measurements for both deployed and undeployed cases are made, we can use methods in [`plot.py`](plot.py) to analyze the results.
By running `plot.py` as follows, a `csv` file with all the measurement results will be saved in [`csv`](csv) and four default plot figures will be saved in the [`figs`](figs) folder:
```
python3 plot.py <label>
```
The `csv` file has the following columns:
```

```

The default figures are:
1. latency vs QPS with jitter
2. latency vs QPS without jitter
3. latency vs number of connections with jitter
4. latency vs number of connections without jitter

The `<label>` input above is optional. If not given, it will default to the label of the
most recently generated json files.

Customized plots can be made by the following command:
```
python3 plot.py <label> <jitter> <param> <default> <percent>
```
Here `<label>` is required, `<jitter>` can be `True` or `False`, `<param>` can
be `SocketCount` or `RequestedQPS`, `<default>` should be the parameter that is
not passed in through `<param>`, and `<percent>` can be `50`, `75`, `90`, `99`,
or `99.9`.


