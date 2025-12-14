# Cloud Run on OrbStack (macOS ARM) - 2025 Guide

<!--
Copyright 2025 Raphaël MANSUY
Licensed under the Apache License, Version 2.0
https://www.apache.org/licenses/LICENSE-2.0
-->

This guide is optimized for running a **True Cloud Run Equivalent** locally on macOS using **OrbStack**. OrbStack is the recommended runtime for 2025 because of its lightweight virtualization, native networking, and seamless Apple Silicon (ARM64) support.

## Architecture Overview

On OrbStack, Knative runs inside a lightweight Linux VM but feels native to macOS.

```ascii
                                    +---------------------------------------------------+
                                    |                 OrbStack VM (Linux)               |
                                    |                                                   |
    [User's Mac Terminal]           |  +---------------------------------------------+  |
          |                         |  |             Kubernetes Cluster              |  |
          |                         |  |                                             |  |
          | 1. curl hello...        |  |   +------------------+      +-----------+   |  |
          |                         |  |   | Envoy (Ingress)  | ---> | Activator |   |  |
          +-----------------------> |  |   | (LoadBalancer)   |      | (Buffer)  |   |  |
             (Native Network)       |  |   +--------+---------+      +-----+-----+   |  |
                                    |  |            |                      |         |  |
                                    |  |            | 2. Route             | 3. Wake |  |
                                    |  |            v                      v         |  |
                                    |  |   +-------------------------------------+   |  |
                                    |  |   |           Pod (Your App)            |   |  |
                                    |  |   |                                     |   |  |
                                    |  |   |  [Queue Proxy] ----> [User Cont.]   |   |  |
                                    |  |   |   (Sidecar)           (Hello)       |   |  |
                                    |  |   +-------------------------------------+   |  |
                                    |  +---------------------------------------------+  |
                                    +---------------------------------------------------+
```

**Key Components:**
1.  **Envoy (Contour)**: The entry point. OrbStack automatically assigns it a reachable IP (accessible from macOS).
2.  **Activator**: If your app is scaled to zero, Envoy sends traffic here to hold the request while your app starts.
3.  **Queue Proxy**: A tiny sidecar in your pod that handles metrics and concurrency limits (just like Cloud Run).

---

## 1. Prerequisites

- **OrbStack** installed and running.
- **Kubernetes** enabled in OrbStack settings.
- **Homebrew** (for installing the `kn` CLI).

```bash
# Install the Knative CLI (client)
brew install knative/client/kn
```

## 2. Install Knative Serving (v1.20.0 - ARM64 Ready)

OrbStack runs standard Kubernetes, so we use the standard YAML manifests. All images are multi-arch and work perfectly on M1/M2/M3 chips.

```bash
# 1. Install CRDs (Custom Resource Definitions)
kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.20.0/serving-crds.yaml

# 2. Install Core Components (Serving)
kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.20.0/serving-core.yaml

# 3. Install Networking Layer (Contour + Envoy)
# Note: We use the 'knative-extensions' repo for net-contour
kubectl apply -f https://github.com/knative-extensions/net-contour/releases/download/knative-v1.20.0/net-contour.yaml

# 4. Configure Knative to use Contour
kubectl patch configmap/config-network \
  -n knative-serving \
  --type merge \
  -p '{"data":{"ingress.class":"contour.ingress.networking.knative.dev"}}'
```

## 3. Configure DNS (Magic DNS)

OrbStack exposes LoadBalancers directly. We need to tell Knative to use "Magic DNS" (`sslip.io`) so you get real URLs like `http://hello.default.192.168.x.y.sslip.io`.

```bash
# Enable "Magic DNS" (sslip.io)
kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.20.0/serving-default-domain.yaml
```

*Wait for components to be ready:*
```bash
kubectl wait --for=condition=Ready pods --all -n knative-serving --timeout=60s
```

## 4. Enable "Cloud Run" Features (Scale-to-Zero)

By default, Knative might be conservative. Let's tune it for that "instant" Cloud Run feel.

```bash
kubectl patch cm config-autoscaler -n knative-serving --type merge -p '{
  "data": {
    "enable-scale-to-zero": "true",
    "scale-to-zero-grace-period": "6s",
    "container-concurrency-target-default": "100"
  }
}'
```

## 5. Deploy Your First Service

Now, deploy a container. We'll use the standard Google "Hello" container.

```bash
kn service create hello \
  --image=us-docker.pkg.dev/cloudrun/container/hello \
  --port=8080 \
  --scale-window=6s
```

**Expected Output:**
```text
Service 'hello' created to latest revision 'hello-00001' is available at URL:
http://hello.default.192-168-215-2.sslip.io
```

> **Note:** The IP address (`192-168-215-2`) is the LoadBalancer IP assigned by OrbStack. You can open this URL directly in Safari or Chrome on your Mac.

## 6. Verify Scale-to-Zero

1.  **Watch the pods** in a separate terminal:
    ```bash
    kubectl get pods -w
    ```
2.  **Wait** about 60 seconds (or whatever the grace period is). You should see the pod **Terminate**.
3.  **Hit the URL** again.
    ```bash
    curl http://hello.default.192-168-xxx-xxx.sslip.io
    ```
4.  **Observe**: The pod will instantly transition to `Pending` -> `ContainerCreating` -> `Running`. The request will complete successfully.

## Troubleshooting OrbStack Specifics

-   **"Connection Refused"**: Ensure the Envoy LoadBalancer has an External IP.
    ```bash
    kubectl get svc -n contour-external envoy
    ```
    It should show an `EXTERNAL-IP` (e.g., `192.168.x.x`). If it says `<pending>`, OrbStack might need a restart, but this is rare.

-   **DNS Issues**: If `sslip.io` is blocked by your corporate VPN, you can use `curl -H "Host: hello.default.example.com" http://<EXTERNAL-IP>` instead.

## Diagnostics & Debugging

The installer script (`knative_orbstack.sh`) includes a `--debug` flag. When enabled, the script will collect additional diagnostic information when testing services. If a service test fails, diagnostics are now automatically saved under `/tmp/knative-diag-<timestamp>-<service>` to help you debug failures quickly.

What is saved:
- `ksvc-<name>.yaml` and `ksvc-<name>.describe` — service definition and Knative status
- `revisions-<name>.yaml` — revision resources
- `pods-<name>.txt` — pod list for the service
- `endpoints-all.yaml` — all endpoints in the cluster (to locate the backends)
- `envoy-svc.yaml`, `envoy-endpoints.yaml` — Envoy service and endpoints
- `envoy-logs.log`, `contour-logs.log` — logs for Envoy and Contour
- `net-contour-controller.log` — net-contour controller logs
- `events-*.txt` — recent events for default, knative-serving, and projectcontour namespaces

Example:

```bash
# Run installer with debug info enabled; this will collect diagnostics if a test fails
./knative_orbstack.sh --debug

# If a test fails the script will print the diagnostics directory location, e.g.
# Diagnostics saved to: /tmp/knative-diag-20251213-153000-hello

# Print a short tail of the logs saved
tail -n 200 /tmp/knative-diag-*/net-contour-controller.log | sed -n '1,200p'
```

