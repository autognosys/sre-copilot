# triage-agent

Go-based K8s controller that watches Pods for known bad-deploy signatures
(`OOMKilled`, `CrashLoopBackOff` today — extend `internal/watcher/watcher.go`
to add more) and, per pod, diagnoses root cause via the Claude API — subject
to a per-pod cooldown and a hard monthly budget circuit breaker
(`internal/costguard/`).

**v1 status:** watch loop + cost guard are wired up and detection-tested.
The actual Claude reasoning step is stubbed (logs what it *would* triage) —
wiring in the tool_use loop is the next pass, once this deploys cleanly and
correctly fires on the known bad-deploy scenario.

## Build

On symphony (or wherever you build):

```bash
cd app/triage-agent
go mod tidy   # resolves the full dependency graph, generates go.sum —
              # go.mod here is hand-written, this is the first real build
docker build -t autognosys/triage-agent:v1 .
docker save autognosys/triage-agent:v1 -o /tmp/triage-agent.tar
```

## Deploy — same no-registry pattern as order-service

```bash
scp /tmp/triage-agent.tar root@65.108.102.155:/tmp/
scp -o ProxyJump=root@65.108.102.155 -i ~/.ssh/sre_copilot_ed25519 /tmp/triage-agent.tar ubuntu@192.168.100.10:/tmp/
# on the VM:
sudo k3s ctr images import /tmp/triage-agent.tar
```

## RBAC, secret, and the Deployment itself

```bash
kubectl apply -f ../../infra/k8s/triage-agent/rbac.yaml

# Claude API key — never committed, same handling as every other credential here
kubectl create secret generic triage-agent-secrets \
  --from-literal=anthropic-api-key='sk-ant-...' \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -f ../../infra/k8s/triage-agent/deployment.yaml
kubectl logs -f deploy/triage-agent
```

## Local dev, outside the cluster

`buildClientset()` in `main.go` falls back to `$KUBECONFIG` when there's no
in-cluster config, so you can run this straight from symphony for faster
iteration than the full build/import/deploy loop:

```bash
export KUBECONFIG=/home/snyshadham/git/sre-copilot/infra/ansible/fetched/sre-copilot-kubeconfig.yaml
go run .
```

(Needs the SSH tunnel to the cluster open, same as `kubectl` — see
`infra/ansible/README.md`.)

## Verifying detection works

Trigger the existing bad-deploy scenario and watch the logs:

```bash
kubectl patch deployment order-service --patch-file ../../infra/k8s/toy-service/bad-deploy-memory-leak.yaml
kubectl logs -f deploy/triage-agent
```

Expect an `"incident detected"` log line (category `oom_kill`) within
seconds of the OOMKill actually happening, followed by either a triage
attempt or a `"skipping — pod in cooldown"` line on subsequent restarts.
