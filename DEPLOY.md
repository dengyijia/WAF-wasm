# Deployment on Istio

We will walk through how to deploy the WAF WASM filter on Istio through the
sample application Bookinfo. Specifically, we will deploy the filter on the
`productpage` service of the Bookinfo application.

There are two pre-requisites:
1. Follow the instructions in README.md to build the
   WASM filter with `proxy-wasm-cpp-sdk`. You should have a
   `WAF_wasm.wasm` file at the root of this repository.
2. Install Istio 1.7.0 and can run the sample Bookinfo
   application following the instructions
   [here](https://istio.io/latest/docs/setup/getting-started/) on istio.io.
   Before starting on the deployment of this WASM filter, clean up the Bookinfo
   application with the command ```samples/bookinfo/platform/kube/cleanup.sh```

Currently, we have two steps in the deployment: mount the `.wasm` file onto
the volume of Istio proxy and patch the configuration of the Istio proxy to use
the filter. In the future release of Istio 1.7.1, there will be tools to complete the
deployment in one step.

### Mount the WASM file to Istio proxy
Create a configuration map on Kubernetes that include the `WAF_wasm.wasm` file (make sure to
modify the path as it is on your local machine).
```
kubectl create cm waf-wasm-filter --from-file=<path-to-WAF-wasm-repo>/WAF_wasm.wasm
```
Using the configuration map, add user volume annotations to the `productpage` deployment of
`samples/bookinfo/platform/kube/bookinfo.yaml` (`productpage` is the last service
configured in the file). The `spec/template/metadata` section of your
`bookinfo.yaml` should look like this:
```
metadata:
  labels:
    app: productpage
    version: v1
  annotations:
    sidecar.istio.io/userVolume: '[{"name":"wasmfilters-dir","configMap": {"name": "waf-wasm-filter"}}]'
    sidecar.istio.io/userVolumeMount: '[{"mountPath":"/var/local/lib/wasm-filters","name":"wasmfilters-dir"}]'
```

### Patch the Istio proxy config for filter
Create a `filter.yaml` file with the following content:
```
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: waf-wasm-filter
  namespace: default
spec:
  configPatches:
    - applyTo: HTTP_FILTER
      match:
        context: SIDECAR_INBOUND
        proxy:
          proxyVersion: '^1\.7.*'
        listener:
          filterChain:
            filter:
              name: envoy.http_connection_manager
              subFilter:
                name: envoy.router
      patch:
        operation: INSERT_BEFORE
        value:
          name: envoy.filters.http.wasm
          typed_config:
            "@type": type.googleapis.com/udpa.type.v1.TypedStruct
            type_url: type.googleapis.com/envoy.extensions.filters.http.wasm.v3.Wasm
            value:
             config:
               name: "WAF_wasm_filter"
               root_id: "root_WAF"
               vm_config:
                 vm_id: "WAF_wasm_filter"
                 runtime: "envoy.wasm.runtime.v8"
                 code:
                   local:
                     filename: "/var/local/lib/wasm-filters/WAF_wasm.wasm"
                 allow_precompiled: true
  workloadSelector:
    labels:
      app: productpage
      version: v1
---
```
The EnvoyFilter defined in `filter.yaml` will patch the contents in
`spec/configPatches/patch/value` to the istio proxy of `productpage-v1` as
specified in `spec/workloadSelector`. Now apply the patch:
```
kubectl apply -f filter.yaml
```

### Run the Bookinfo application
Now you can run the Bookinfo application with the instructions on istio.io.
There are several things you can do to verify that the filter has been deployed.

First, check that the `WAF_wasm.wasm` file was mounted correctly in the Istio
proxy volume:
```
kubectl exec -it deployment/productpage-v1 -c istio-proxy -- ls
/var/local/lib/wasm-filters/
```
Then, check that the filter configuration has been patched to the Istio proxy configuration in the pod of `productpage`:
```
istioctl proxy-config listeners productpage-v1-<pod-name> -o json | grep
"WAF_wasm_filter"
```
The return should not be empty. Repeat these two steps on other services such as
`reviews` or `ratings` to verify that the filter was only deployed on
`productpage` and none of the others.

Most importantly, after you have export the ingress gateway URL, you can
directly interact with the `productpage`:
```
curl -s "http://${GATEWAY_URL}/productpage" -v
curl -s "http://${GATEWAY_URL}/productpage" -v --cookie "val=1' UNION SELECT
user"
```
The first request should receive the HTML of the product page, while the
second request should be blocked from the server with a response of HTTP 403
Forbidden. In both responses, an additional header `filter: WAF_wasm` should be
present.
