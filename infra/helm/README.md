# infra/helm (+ infra/k8s/openobserve)

OTel Collector via Helm, OpenObserve via a plain manifest. Run these with
the SSH tunnel from `infra/ansible/README.md` open (or `KUBECONFIG` pointed
at a kubeconfig that can already reach the cluster).

## Why OpenObserve isn't on Helm here

The `openobserve/openobserve` chart defaults to a distributed HA topology —
separate ingester/querier/router/scheduler/compactor pods, a 3-replica NATS
cluster, and a 2-instance Postgres that requires the `cloudnative-pg`
operator installed separately. That's ~12 pods plus an extra operator
dependency — the wrong fit for a single 16GB lab node, and it fails outright
without the operator.

`ZO_NODE_ROLE=all` is OpenObserve's own documented single-binary deployment
path (same shape as their quickstart `docker run`) — one process does
ingestion, querying, and routing. `infra/k8s/openobserve/openobserve.yaml`
runs that directly as a Deployment + PVC + Service, no chart involved.

## 1. Namespace

```bash
kubectl create namespace observability
```

## 2. Real credentials, not the placeholder in the manifest

The manifest's Secret ships a throwaway placeholder password so
`kubectl apply -f` doesn't hard-fail on its own. Set the real one first —
this `kubectl create secret` takes precedence over the placeholder when
applied after:

```bash
export OO_ROOT_PASSWORD='choose-something-here'
kubectl create secret generic openobserve-auth \
  -n observability \
  --from-literal=ZO_ROOT_USER_EMAIL=admin@sre-copilot.local \
  --from-literal=ZO_ROOT_USER_PASSWORD="$OO_ROOT_PASSWORD" \
  --dry-run=client -o yaml | kubectl apply -f -
```

## 3. Deploy OpenObserve

```bash
kubectl apply -f ../k8s/openobserve/openobserve.yaml
kubectl get pods -n observability -l app=openobserve
```

## 4. OTel Collector — check chart values before installing

Chart value keys drift across versions — do this before every install:

```bash
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update
helm show values open-telemetry/opentelemetry-collector > /tmp/otel-defaults.yaml
```

Diff `values-otel-collector.yaml` in this directory against that — top-level
keys (`mode`, `config`, `ports`, `resources`, `replicaCount`) were confirmed
to match on 2026-09-18; re-check if the chart version has moved since.

## 5. Wire up the OTLP exporter auth header

```bash
echo -n "admin@sre-copilot.local:$OO_ROOT_PASSWORD" | base64
```

Paste that into `values-otel-collector.yaml`'s
`exporters.otlphttp/openobserve.headers.Authorization` in place of
`REPLACE_WITH_BASE64_USER_PASS` — or override at install time:

```bash
helm install otel-collector open-telemetry/opentelemetry-collector \
  -n observability \
  -f values-otel-collector.yaml \
  --set-string 'config.exporters.otlphttp/openobserve.headers.Authorization'="Basic $(echo -n "admin@sre-copilot.local:$OO_ROOT_PASSWORD" | base64)"
```

## 6. Verify

```bash
kubectl get pods -n observability
kubectl logs -n observability deploy/otel-collector-opentelemetry-collector
```

Port-forward OpenObserve's UI to check ingestion once the toy app (see
`app/toy-service/`) is sending traces:

```bash
kubectl port-forward -n observability svc/openobserve 5080:5080
# then browse http://localhost:5080 (through the existing SSH tunnel setup
# if browsing from a machine other than the one running kubectl)
```
