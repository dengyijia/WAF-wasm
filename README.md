# WAF extension on Envoy proxy

This repository is forked from [`envoyproxy/envoy-wasm`](https://github.com/envoyproxy/envoy-wasm), and the example WASM extension in the envoy-wasm repository is modified to work as a Web Application Firewall(WAF) that can detect SQL injection. The rules for detection are aligned with ModSecurity rules [942100](https://github.com/coreruleset/coreruleset/blob/v3.3/dev/rules/REQUEST-942-APPLICATION-ATTACK-SQLI.conf#L45) and [942101](https://github.com/coreruleset/coreruleset/blob/v3.3/dev/rules/REQUEST-942-APPLICATION-ATTACK-SQLI.conf#L1458), and SQL injection is detected with methods from [libinjection](https://github.com/client9/libinjection).

## Build

We will build the WASM module with `proxy-wasm-cpp-sdk` on docker.
We first need to build the image of the SDK from its repository on the `envoy-release/v1.15` branch:
```
git clone git@github.com:proxy-wasm/proxy-wasm-cpp-sdk.git
cd proxy-wasm-cpp-sdk
git checkout envoy-release/v1.15
docker build -t wasmsdk:v2 -f Dockerfile-sdk .
```
Then from the root of this repository, build the WASM module with:
```
docker run -v $PWD:/work -w /work wasmsdk:v2 /build_wasm.sh
```
After the compilation completes, you should find a `WAF_wasm.wasm` file in the repository.

## Deploy
We can mount the WASM module onto the docker image of Istio proxy to run it.
Pull the following image of Istio proxy:
```
docker pull istio/proxyv2:1.7.0-beta.2
```
Then run the proxy with WASM configured:
```
docker run \
-v ${PWD}/envoy.yaml:/etc/envoy.yaml \
-v ${PWD}/WAF_wasm.wasm:/etc/WAF_wasm.wasm \
-p 8000:8000 \
--entrypoint /usr/local/bin/envoy \
istio/proxyv2:1.7.0-beta.2 -l trace --concurrency 1 -c /etc/envoy.yaml
```

In a separate terminal, curl at `localhost:8000` to interact with the running proxy. For example, if you type the following command, you will receive a response with HTTP code 200 Okay, indicating that the request has passed SQL injection detection.
```
curl -d "hello world" -v localhost:8000
```
If you instead put something suspicious in the body, for example, enter the following command:
```
curl -d "val=-1%27+and+1%3D1%0D%0A" -v localhost:8000
```
You will receive a response with HTTP code 403 Forbidden. The body of the http request above has the parameter `val` with the value `-1' and 1=1` in URL
encoding.

## Unit Tests

Unit tests for individual utility functions in the WAF WASM extension are
available in `test` directory. To run them, execute from the root of the
repository:
```
source ./build_test.sh
```


## Configuration
Users of the filter can decide which parts of http requests should go through SQL injection detection by passing in configuration strings. Specifically, the configuration should be passed in through the field `config.config.configuration.value` in JSON syntax in envoy configuration YAML files. An example can be found in `envoy-config.yaml`:
```
{
  “body”: {
    # detect sqli on all parameters but “foo”
    “Content-Type”: “application/x-www-form-urlencoded”,
    “exclude”: [“foo”]
  },
  “header”: {
    # detect sqli on “bar”
    “include”: [“bar”]
  }
}
```

There are four parts that can be configured: query parameters in body(`body`),
query parameters in path(`path`), cookies(`cookie`), and headers(`header`).
Configuration for all four parts are optional. If an `include` is populated for
a part, the WAF filter will only inspect the fields corresponding to the given
keys. If an `exclude` is populated, the WAF filter will inspect all but the
given keys. If nothing is passed for a part, a default configuration based on
ModSecurity rule 942100 will apply. ModSecurity rule 942101 requires SQL
injection detection on the entire path, all cookie names and values, all query
parameters in body, and two header fields "User-Agent" and "Referer". `include
and `exclude` are not expected to be present at the same time.

If `body` is present in the configuration, the "Content-Type" field is required. Currently, the WASM filter only supports SQL injection detection for the content type "application/x-www-form-urlencoded" (it has the syntax `param=value&param2=value2`). If the incoming http request has a different content type, detection on its body will be skipped.

## Indentation
Run `make indent` to format all C++ files.
