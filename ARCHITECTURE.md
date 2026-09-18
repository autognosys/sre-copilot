# sre-copilot — Architecture & Scope

## Why this exists

I run OpenTelemetry + OpenObserve + Kubernetes in my own homelab, and I kept coming back to the same question: the current wave of "AI-SRE" tools (Resolve.ai, Cleric, Datadog Bits, and others) are genuinely useful as assistants, but the honest evidence is that they're not yet trustworthy as autonomous fixers. Human-in-the-loop is still the winning pattern.

That raised a narrower, more answerable question: **can a small, honestly-calibrated agent reliably diagnose one specific, common class of incident — and know when it doesn't know?**

This project is an attempt to answer that on real infrastructure, built to production quality rather than demo quality, on a slice narrow enough to actually finish and evaluate properly.

## What it does

When a bad deploy breaks something in the watched Kubernetes cluster, the agent:

1. Detects the anomaly (error rate spike, crash loop, OOM kill)
2. Correlates it against recent changes (deploy timestamps, manifest diffs)
3. Walks the dependency chain where relevant
4. Ranks hypotheses by likelihood
5. Posts an evidence-cited root-cause diagnosis, with a calibrated confidence score

It is **read-only** — it investigates and reports. It does not remediate anything automatically. That's a deliberate choice, not a limitation I ran out of time to fix: autonomous remediation is a much harder trust problem, and bolting it on would undercut the actual point of this project, which is to be honestly useful within a narrow, well-understood boundary.

## Incident class: bad deploy / recent change

This is the single incident class v1 targets. Concretely, the injected faults look like:

- A manifest change with a bad or missing image tag → `ImagePullBackOff`
- A manifest change that breaks a health check or required config → `CrashLoopBackOff`
- A manifest change that sets resource limits too low → immediate `OOMKilled` right after rollout

Why this one: it's arguably the single most common real-world incident class, it has a clean, boundable investigation shape (what changed, when did things break, do the timestamps line up), and it's fully reproducible with a `kubectl apply` of a broken manifest — which makes it something I can evaluate rigorously rather than just demo.

Other incident classes (resource exhaustion unrelated to a deploy, downstream dependency failures) were considered and deliberately deferred — see Roadmap below.

## What "correctly diagnosed" means

A diagnosis counts as correct if, within a fixed time window of the injected fault, it produces:

1. **The right root-cause category** — not just "something is wrong," but "deployment X introduced Y"
2. **A cited evidence chain** — the specific change (commit / image tag / manifest diff timestamp) tied to the specific signal (error spike, log pattern, OOM event, metric threshold) that connects cause to effect
3. **A calibrated confidence score** — and just as importantly, a *low* confidence score on a case it genuinely can't solve. An agent that's always confident isn't calibrated, it's just noisy.

This is validated against a fixed eval suite of injected incidents, including one deliberately out-of-scope case designed to check that the agent says "I don't know" instead of hallucinating a plausible-sounding wrong answer. The eval results (including the case it honestly punts on) are published alongside the code — see `/eval`.

## Architecture

```
┌─────────────────────────────────────────────┐
│  k3s cluster (bare metal, on-prem)           │
│  ┌─────────────┐                             │
│  │  Workloads   │──── faults injected here   │
│  └──────┬──────┘                             │
│         │ telemetry                          │
│  ┌──────▼──────┐                             │
│  │ OTel         │                             │
│  │ Collector    │                             │
│  └──────┬──────┘                             │
│         │                                     │
│  ┌──────▼──────┐                             │
│  │ OpenObserve  │  (logs, metrics, traces)   │
│  └──────┬──────┘                             │
└─────────┼─────────────────────────────────────┘
          │ queried by
   ┌──────▼──────┐
   │  Triage      │  correlation → dependency   │
   │  Agent       │  walk → hypothesis ranking  │
   └──────┬──────┘  → confidence scoring         │
          │
   ┌──────▼──────┐
   │  Notifier    │  (Slack by default,
   │  interface   │   swappable for air-gapped
   └─────────────┘   environments)
```

**Stack:**
- **k3s** — lightweight Kubernetes, running on bare-metal Proxmox
- **OpenTelemetry Collector** — telemetry ingestion
- **OpenObserve** — storage and query layer for logs, metrics, and traces
- **The agent** — reads from OpenObserve, runs the investigation pipeline, posts findings

**Infrastructure as code:** Terraform (Proxmox provider) for VM provisioning, Ansible for cluster bootstrap and service deployment. Boring, well-documented tools, on purpose.

## Design principle: self-hostable by default

Everything in the core path is designed to run without a cloud dependency:

- No hard dependency on a SaaS LLM API — the agent runs against a local model by default
- OpenTelemetry and OpenObserve are both fully self-hostable
- Notifications go through a swappable interface — Slack is the default, but the core logic doesn't know or care what the sink is

This wasn't an afterthought bolted on for a checkbox — it was a day-one constraint, because retrofitting "can this run without the internet" after the fact is far more expensive than designing for it up front.

## What's explicitly out of scope for v1

This is a narrow slice, on purpose. Named here so it's a roadmap, not a gap someone has to guess at:

- **Automatic remediation** — read-only by design, see above
- **Multi-tenancy, RBAC, SSO, SOC2** — enterprise surface area that doesn't belong in a v1 proof of concept
- **A cost-optimizing model router** (cheap model for triage, escalate to a bigger model for hard cases) — a natural v1.1, deliberately deferred to keep v1 finishable
- **Cluster autoscaling** — not really a coherent concept on bare metal without a cloud provider's node API; the honest on-prem answer is pre-provisioned capacity or a manual node-join runbook, not fake automation
- **Incident classes beyond "bad deploy"** — resource exhaustion and downstream dependency failures are good v2 candidates once this pipeline is proven

## Roadmap

- [ ] v1: bad-deploy incident class, single local model, Slack output, fixed eval suite
- [ ] v1.1: cheap/expensive model router for cost-aware escalation
- [ ] v2: additional incident classes (resource exhaustion, downstream dependency failure)
- [ ] Someday: exploring what a safe, bounded remediation action would even look like (this is a hard trust problem and isn't being rushed)
