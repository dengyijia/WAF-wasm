# WAF WASM filter on Envoy proxy

This repository contains the source code for a WebAssembly Module(WASM) that can work as a Web Application Firewall(WAF) on envoy proxy. The filter parses incoming http requests to the proxy. If a SQL injection attack is detected, the request will be blocked from upstream servers.

The rules for SQL injection detection are aligned with ModSecurity rules [942100](https://github.com/coreruleset/coreruleset/blob/v3.3/dev/rules/REQUEST-942-APPLICATION-ATTACK-SQLI.conf#L45) and [942101](https://github.com/coreruleset/coreruleset/blob/v3.3/dev/rules/REQUEST-942-APPLICATION-ATTACK-SQLI.conf#L1458), and strings with potential SQL injection attacks are parsed with methods from [libinjection](https://github.com/client9/libinjection).

## Build

We will build the WASM filter with `proxy-wasm-cpp-sdk` on docker.
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
After the compilation completes, you should find a `WAF_wasm.wasm` file in the repository directory.

## Deploy
We can mount the WASM filter onto the docker image of Istio proxy to run it.
Pull the following image of Istio proxy:
```
docker pull istio/proxyv2:1.7.0-beta.2
```
Then run the proxy with WASM configured:
```
docker run \
-v ~/WAF-wasm/envoy.yaml:/etc/envoy.yaml \
-v ~/WAF-wasm/WAF_wasm.wasm:/etc/WAF_wasm.wasm \
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
./unit_test.sh
```

## Integration Tests

Before conducting integration tests, make sure that the WASM binary file has been built and the `istio/proxy` docker image has been pulled according to instructions in previous sections. The tests can be run by executing the following script from the root of
the repository:
```
./integration_test.sh
```
In the integration tests, the WASM filter is configured onto Istio proxy. The
proxy receives messages from an http client and forwards them to an http server.
Both the client and the server are set up in GoLang. We monitor all
communications between the client, proxy, and server to see if the WASM filter
is working as expected.

The source code of the integration tests are in `integration_test`:
`integration_test.go` contains the test framework that handles the sending and
receiving of requests, and `main.go` contains the input and expectations of
specific test cases.

## Configuration
The rules for SQL injection detection can be configured from YAML files. An example of configuration can be found in `envoy-config.yaml`. Configuration are passsed through the field `config.config.configuration.value` in the yaml file in JSON syntax as below:

```
{
  “query_param”: {
    # detect sqli on all parameters but “foo”
    “Content-Type”: “application/x-www-form-urlencoded”,
      “exclude”: [“foo”]
  },
    “header”: {
      # detect sqli on “bar”, “Referrer”, and “User-Agent”
      “include”: [“bar”]
    }
}
```

There are three parts that can be configured for now: query parameters(`query_param`), cookies(`cookie`, not shown above), and headers(`header`). Configuration for all three parts are optional. If nothing is passed in a field, a default configuration based on ModSecurity rule 942100 will apply. ModSecurity rule 942101 requires SQL injection detection on path of request. Configuration for path will be updated later.

### Query Parameters
The "Content-Type" field is required in query parameters configuration, Currently, the WASM module only supports SQL injection detection for the content type "application/x-www-form-urlencoded" (it has the syntax `param=value&param2=value2`). If the incoming http request has a different content type, detection on its body will be skipped.

In default setting, all query parameter namesand values will be checked for SQL injection. To change this setting, you can either add an `include` or an `exclude` field. Both take a list of parameter names. If `include` is present, only the parameters in the list will be checked. If `exclude` is present, all but the parameters in the list will be checked. `include` and `exclude` are not expected to be present at the same time.

### Headers
In default setting, the `Referrer` and `User-Agent` headers will be checked for SQL injection. The `include` and `exclude` fields work similarly as above, except that `Referrer` and `User-Agent` will always be checked unless explicitly enlisted in `exlude`.

### Cookies
In default setting, all cookie names will be checked. `include` and `exclude` work exactly the same as for query parameters.

