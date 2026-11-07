# KubeWhy — Feature Ideas & Roadmap

## Overview

The current KubeWhy concept is strong: a deterministic Kubernetes troubleshooting tool that turns scattered Kubernetes resource state into an evidence-backed explanation of **why something is broken**, rather than simply dumping `kubectl describe` output.

The main opportunity is to expand from:

> **Why is this Pod broken?**

to:

> **Why is my Kubernetes application or system broken?**

The guiding principle should remain:

**evidence → dependency graph → root cause → explanation**

Avoid making an LLM the core diagnostic engine. An optional AI explanation layer can be added later while keeping the underlying diagnosis deterministic and auditable.

---

# 1. Service Diagnosis

### Command

```bash
kubewhy service api
```

### Example

```text
✗ Service/api is not receiving traffic

ROOT CAUSE
  Service has no ready endpoints.

Evidence
  selector: app=api
  matching pods: 3
  ready pods: 0

  Pod/api-7d8f...
    readiness probe failing: HTTP 503

  Pod/api-91ab...
    CrashLoopBackOff

  Pod/api-a21...
    Pending: insufficient memory

Likely impact
  Requests to api will fail or time out.
```

### Checks

- Service selector → matching Pods
- EndpointSlices
- Ready vs unready endpoints
- `targetPort`
- Named ports
- Service type
- Headless Services
- Selector mismatches
- Endpoints disappearing because of readiness
- Missing target Pods

### Why

This is a natural extension of Pod diagnosis because KubeWhy already has much of the information required to diagnose the next layer of the Kubernetes networking stack.

---

# 2. Deployment Diagnosis

### Command

```bash
kubewhy deployment api
```

Instead of diagnosing individual Pods, explain **why the Deployment isn't achieving its desired state**.

### Example

```text
Deployment/api is unhealthy

Desired replicas: 10
Available: 7
Ready: 7

ROOT CAUSE
  3 newly-created Pods cannot start.

  2 → ImagePullBackOff
  1 → insufficient CPU on available nodes

Rollout:
  7 old Pods: healthy
  3 new Pods: failing
```

### Checks

- Rollout status
- `ProgressDeadlineExceeded`
- Desired vs available replicas
- Ready vs unavailable replicas
- `maxSurge`
- `maxUnavailable`
- ReplicaSet mismatch
- Old ReplicaSets still serving traffic
- New image failures
- Pods sharing the same underlying failure

### Important behavior

If 20 Pods have the same failure, don't produce 20 separate diagnoses.

Instead:

```text
18 Pods are failing because the new container image cannot be pulled.
```

This aggregation should become a general KubeWhy design principle.

---

# 3. Dependency-Aware Root Cause Analysis

This could become the defining feature of KubeWhy.

### Example

```bash
kubewhy service checkout
```

Instead of stopping at:

```text
Service has no endpoints
```

follow the dependency chain.

```text
Service
  ↓
EndpointSlice
  ↓
Pod
  ↓
Container
  ↓
Readiness probe
  ↓
Application dependency
```

### Example output

```text
Service/checkout is unavailable

ROOT CAUSE

checkout → Pod → readiness probe → application

The checkout Pods are running but aren't Ready.

The readiness probe returns HTTP 503.

Application evidence:
  connection refused: postgres:5432

Postgres Service has no ready endpoints.

ROOT CAUSE:
  postgres Pods are failing with CrashLoopBackOff.
```

### Goal

KubeWhy should progressively answer:

1. What is broken?
2. What directly caused it?
3. What caused that?
4. What is the earliest/root cause?
5. What is the impact?

This is more valuable than simply reporting symptoms.

---

# 4. Node Diagnosis

### Command

```bash
kubewhy node worker-3
```

### Checks

- MemoryPressure
- DiskPressure
- PIDPressure
- NotReady
- Cordoned nodes
- Unschedulable nodes
- Node taints
- Node conditions
- Resource capacity
- Resource allocation
- Evicted Pods
- Scheduling failures

### Example

```text
Deployment/api cannot schedule 3 replicas.

The scheduler rejected all 5 available nodes.

Reasons:

  worker-1 → insufficient memory
  worker-2 → taint: dedicated=gpu
  worker-3 → MemoryPressure
  worker-4 → insufficient memory
  worker-5 → nodeSelector mismatch
```

The important part is connecting node-level problems back to workload-level impact.

---

# 5. Ingress Diagnosis

### Command

```bash
kubewhy ingress api
```

### Dependency chain

```text
Ingress
 ↓
Ingress rule
 ↓
Service
 ↓
Service port
 ↓
EndpointSlice
 ↓
Pod
```

### Example

```text
✗ Ingress/api is unable to route traffic

Host:
  api.example.com

Path:
  /v1

Ingress → Service
  api:443

Service → targetPort
  8080

Service → Endpoints
  none

ROOT CAUSE
  No ready Pods match the Service selector.
```

### Checks

- Incorrect Service
- Incorrect Service port
- Incorrect targetPort
- Missing backend
- No endpoints
- No ready endpoints
- TLS Secret missing
- Invalid/missing IngressClass
- Ingress controller not observing the Ingress

---

# 6. PVC / Storage Diagnosis

### Command

```bash
kubewhy pvc postgres-data
```

### Example

```text
PVC/postgres-data is not usable

Status:
  Pending

StorageClass:
  gp3

ROOT CAUSE
  No PersistentVolume can satisfy the request.

Requested:
  500Gi
  ReadWriteOnce

Available:
  100Gi max

The PVC has been pending for 14m.
```

### Dependency chain

```text
Pod
 ↓
PVC
 ↓
PV
 ↓
StorageClass
 ↓
Provisioner
```

### Checks

- Pending PVC
- Missing PV
- StorageClass issues
- Provisioning failures
- Capacity mismatch
- Access mode mismatch
- Volume mode mismatch
- Binding problems
- Volume attachment problems

---

# 7. Cluster / Namespace-Wide Scan

### Command

```bash
kubewhy scan
```

or:

```bash
kubewhy why
```

### Example

```text
KUBERNETES STATUS
────────────────────────────────────

3 incidents detected

1. checkout
   Deployment unavailable
   └── Pods failing readiness
       └── postgres Service has no endpoints
           └── postgres Pods OOMKilled

2. payments
   Deployment rollout stuck
   └── ImagePullBackOff
       └── image tag does not exist

3. ingress
   api.example.com returning 503
   └── api Service has no ready endpoints
```

### Critical feature: deduplication

If 40 resources are affected by one root cause, report:

```text
1 root cause
40 affected resources
```

not 40 independent errors.

---

# 8. Flagship `kubewhy why`

A possible flagship UX:

```bash
kubewhy why
```

Meaning:

> What's wrong with this cluster?

KubeWhy investigates unhealthy resources and builds a dependency graph.

### Example

```text
CLUSTER STATUS
────────────────────────────────────

✗ checkout       Deployment
✗ checkout       Service
✗ postgres       StatefulSet
✗ postgres       Service

ROOT CAUSE

postgres StatefulSet
└── postgres-0
    └── OOMKilled
        └── memory limit: 512Mi

IMPACT

postgres Service has 0 ready endpoints
    ↓
checkout Pods fail readiness
    ↓
checkout Service has 0 ready endpoints
    ↓
Ingress returns 503
```

This could become the primary demonstration of KubeWhy.

---

# 9. Event Timeline

### Command

```bash
kubewhy events checkout
```

Do not simply print Kubernetes Events. Correlate them into a timeline.

### Example

```text
14:01  Deployment updated
14:01  ReplicaSet created
14:02  Pod scheduled
14:02  Image pulled
14:03  Container started
14:03  Readiness probe failed
14:04  Container restarted
14:04  CrashLoopBackOff
14:09  Service lost final ready endpoint
```

Then identify likely triggering events:

```text
Likely triggering event:
Deployment rollout at 14:01
```

---

# 10. Change Detection / "What Changed?"

Potential commands:

```bash
kubewhy diff deployment api
```

or:

```bash
kubewhy why --since 1h
```

### Example

```text
api became unhealthy approximately 17 minutes ago.

Changes detected:

Deployment/api

  image:
    v1.42.1 → v1.43.0

  memory limit:
    1Gi → 512Mi

The unhealthy Pods were created after this rollout.
```

### Possible sources

- Kubernetes Events
- ReplicaSets
- `managedFields`
- Helm
- Argo CD
- Git
- Kubernetes object history

This could eventually turn KubeWhy from a static diagnostic tool into an incident investigation tool.

---

# 11. HPA Diagnosis

### Command

```bash
kubewhy hpa api
```

### Example

```text
HPA/api isn't scaling as expected.

Desired replicas: 10
Current replicas: 3

CPU target: 70%
Current CPU: 91%

ROOT CAUSE

HPA cannot obtain CPU metrics.

Metrics API:
  unavailable

Therefore the HPA cannot calculate desired replicas.
```

### Other cases

- `maxReplicas` reached
- `minReplicas` preventing scale-down
- Missing resource requests
- Metrics unavailable
- Invalid target metric
- HPA desired replicas > Deployment capacity
- Scaling blocked by workload constraints

---

# 12. Network Diagnosis

### Command

```bash
kubewhy network api
```

Initially this can be static analysis rather than active network testing.

Use:

- Services
- EndpointSlices
- NetworkPolicies
- Namespace selectors
- Pod selectors
- Ports
- Ingress rules
- Egress rules

### Example

```text
Pod/frontend cannot reach Service/backend

frontend:
  namespace: web
  labels:
    app=frontend

backend:
  namespace: api
  port: 8080

NetworkPolicy/backend:
  ingress:
    namespaceSelector:
      team=payments

frontend namespace:
  team=web

ROOT CAUSE

The NetworkPolicy does not allow:

frontend → backend:8080
```

This could become a particularly powerful Kubernetes troubleshooting feature.

---

# 13. DNS Diagnosis

### Command

```bash
kubewhy dns service/backend
```

### Example

```text
backend.web.svc.cluster.local

Service exists: ✓
ClusterIP: ✓
Endpoints: ✗

DNS itself is unlikely to be the root cause.

The hostname resolves to a Service with zero ready endpoints.
```

The key idea is to explicitly distinguish **symptoms from causes**.

For example, if an application reports:

```text
DNS lookup failed
```

KubeWhy should determine whether:

- CoreDNS is actually broken
- the Service exists
- the Service has endpoints
- the Pod can reach DNS
- the hostname is malformed
- the Service simply has no healthy backends

---

# 14. StatefulSet Diagnosis

### Command

```bash
kubewhy statefulset postgres
```

### Checks

- Ordered rollout
- Replica availability
- Per-replica health
- PVC per replica
- Headless Service
- Pod identity
- Individual replica failures
- Stuck updates
- Volume attachment problems
- Readiness failures

### Example

```text
StatefulSet/postgres is unhealthy

3 replicas desired
2 ready

postgres-2:
  Pod: Running
  Ready: false

PVC:
  postgres-data-postgres-2
  Bound

ROOT CAUSE:
  readiness probe failing

postgres-0 and postgres-1 are healthy.
```

---

# 15. General Dependency Graph

The above features should eventually converge into a common internal model.

Possible relationships:

```text
Ingress
  ↓
Service
  ↓
EndpointSlice
  ↓
Pod
  ↓
Container
  ↓
Probe

Deployment
  ↓
ReplicaSet
  ↓
Pod

StatefulSet
  ↓
Pod
  ↓
PVC
  ↓
PV
  ↓
StorageClass

Pod
  ↓
Node

Pod
  ↓
NetworkPolicy

HPA
  ↓
Deployment
  ↓
Pod
```

KubeWhy can use this graph to recursively investigate failures.

---

# 16. Root Cause vs Symptom

A central design principle should be:

> **Don't report the first error you find as the root cause.**

Example:

```text
Ingress returns 503
    ↓
Service has no endpoints
    ↓
Pods are not Ready
    ↓
Readiness probe returns 503
    ↓
Application cannot connect to PostgreSQL
    ↓
PostgreSQL Pods are OOMKilled
```

The final diagnosis should identify:

```text
ROOT CAUSE
PostgreSQL Pods are being OOMKilled.

SYMPTOMS
- PostgreSQL Service has no ready endpoints
- Checkout Pods fail readiness
- Checkout Service has no endpoints
- Ingress returns 503
```

This distinction is one of the strongest potential differentiators for KubeWhy.

---

# 17. Evidence-Based Output

Every diagnosis should ideally provide evidence.

For example:

```text
ROOT CAUSE
Pod/api-7f8d is failing because its readiness probe returns HTTP 503.

Evidence:
  Pod status: Running
  Ready: false
  Restart count: 0
  Probe:
    HTTP GET /health:8080
  Last probe result:
    HTTP 503
```

This keeps the tool trustworthy.

Avoid:

```text
The application is probably broken.
```

Prefer:

```text
The Pod is not Ready because the configured readiness probe
returned HTTP 503.
```

---

# 18. Suggested CLI Structure

A possible CLI could evolve into:

```bash
kubewhy pod <name>
kubewhy service <name>
kubewhy deployment <name>
kubewhy statefulset <name>
kubewhy daemonset <name>
kubewhy node <name>
kubewhy ingress <name>
kubewhy pvc <name>
kubewhy hpa <name>
kubewhy network <name>
kubewhy dns <name>

kubewhy why
kubewhy scan
kubewhy events <name>
kubewhy diff <resource>
```

Potential global options:

```bash
kubewhy why -n production
kubewhy why --all-namespaces
kubewhy why --since 1h
kubewhy why --output json
kubewhy why --output yaml
kubewhy why --verbose
```

---

# 19. JSON Output

Machine-readable output would make KubeWhy much more useful in automation.

Example:

```json
{
  "resource": "deployment/checkout",
  "status": "unhealthy",
  "rootCause": {
    "type": "PodCrashLoop",
    "resource": "pod/postgres-0",
    "reason": "OOMKilled"
  },
  "impact": [
    "service/postgres has no ready endpoints",
    "deployment/checkout has 0 ready replicas",
    "ingress/checkout is returning 503"
  ]
}
```

This opens the door to:

- CI/CD integrations
- Slack bots
- dashboards
- automated incident tooling
- GitHub Actions
- Argo workflows
- monitoring integrations

---

# 20. Exit Codes

Useful for CI/CD and automation:

```text
0 → healthy / no issues
1 → warnings
2 → unhealthy resources
3 → critical/root-cause failure
```

Potentially:

```bash
kubewhy why --exit-code
```

This would allow:

```bash
kubewhy why -n production || exit 1
```

---

# 21. Watch Mode

Potential command:

```bash
kubewhy why --watch
```

Example:

```text
13:02:01  ✓ cluster healthy

13:02:13  ✗ deployment/api rollout started

13:02:17  ✗ pod/api-7f8d ImagePullBackOff

13:02:18  ✗ service/api now has 0 ready endpoints

13:02:18  ✗ ingress/api likely unavailable
```

This could provide a lightweight incident-monitoring mode.

---

# 22. Optional Integrations

Keep integrations optional rather than making them core dependencies.

Potential integrations:

### Helm

```bash
kubewhy why --helm
```

Use release metadata to identify changes.

### Argo CD

```bash
kubewhy why --argocd
```

Correlate:

```text
Argo CD sync
    ↓
Deployment change
    ↓
Pod failures
```

### Git

Correlate Kubernetes changes with commits:

```text
Deployment changed after commit:

a83f19c
"Increase API memory limit"
```

### Prometheus

Potentially use metrics as additional evidence:

```text
Pod restarted 18 times.

Memory usage:
  508Mi / 512Mi

OOMKilled:
  7 times
```

These should enhance diagnosis rather than replace Kubernetes-native evidence.

---

# 23. Don't Rush Into an LLM

A tempting direction is:

```text
Kubernetes
   ↓
LLM
   ↓
"Here's what's wrong"
```

I would avoid making this the core architecture.

Instead:

```text
Kubernetes
   ↓
Deterministic analyzers
   ↓
Evidence
   ↓
Dependency graph
   ↓
Root cause
   ↓
Structured diagnosis
   ↓
Optional LLM explanation
```

This gives KubeWhy a strong foundation because the actual diagnosis can be:

- deterministic
- reproducible
- testable
- explainable
- auditable

An LLM could later provide:

```bash
kubewhy explain deployment/api
```

without being responsible for deciding what actually happened.

---

# 24. Potential Differentiator

There are already Kubernetes troubleshooting tools and projects that use AI/LLM agents.

KubeWhy could differentiate by focusing on:

> **Evidence-backed, dependency-aware Kubernetes root-cause analysis.**

The core promise:

```text
KubeWhy doesn't just tell you what is unhealthy.

It explains why.

And it shows the evidence behind the explanation.
```

---

# 25. Recommended Roadmap

## Phase 1 — Expand Resource Coverage

### High priority

- [ ] Service analyzer
- [ ] Deployment analyzer
- [ ] Node analyzer
- [ ] Ingress analyzer
- [ ] StatefulSet analyzer

### Medium priority

- [ ] HPA analyzer
- [ ] DNS analyzer
- [ ] NetworkPolicy analyzer
- [ ] Improved PVC/storage analyzer

---

## Phase 2 — Build the Dependency Engine

- [ ] Resource relationship graph
- [ ] Recursive dependency traversal
- [ ] Root-cause detection
- [ ] Symptom vs root-cause classification
- [ ] Evidence collection
- [ ] Failure aggregation
- [ ] Duplicate failure deduplication
- [ ] Impact analysis

This is arguably the most important architectural phase.

---

## Phase 3 — Build the Flagship UX

- [ ] `kubewhy why`
- [ ] `kubewhy scan`
- [ ] Cluster-wide health summary
- [ ] Namespace-wide health summary
- [ ] Root-cause trees
- [ ] Dependency visualization
- [ ] Human-readable incident summaries

---

## Phase 4 — Historical Investigation

- [ ] Event timelines
- [ ] `--since`
- [ ] Change detection
- [ ] ReplicaSet history
- [ ] Managed fields
- [ ] Helm integration
- [ ] Argo CD integration
- [ ] Git correlation

---

## Phase 5 — Automation

- [ ] JSON output
- [ ] YAML output
- [ ] Stable exit codes
- [ ] Watch mode
- [ ] CI/CD usage
- [ ] Slack/ChatOps integrations
- [ ] Monitoring integrations

---

## Phase 6 — Optional AI Layer

- [ ] `kubewhy explain`
- [ ] Natural-language incident summaries
- [ ] Interactive investigation
- [ ] Suggested next diagnostic commands
- [ ] LLM-assisted explanation of deterministic findings

The AI layer should consume KubeWhy's structured evidence rather than directly making unsupported guesses against the cluster.

---

# 26. Feature Priority

| Priority | Feature | Value |
|---|---|---|
| 🔥 Critical | Service analyzer | Natural extension of Pod diagnosis |
| 🔥 Critical | Deployment analyzer | Aggregates Pod failures |
| 🔥 Critical | Dependency/root-cause graph | Potential defining feature |
| 🔥 Critical | `kubewhy why` | Strong primary UX |
| 🔥 Critical | Cluster scan | Excellent operational workflow |
| High | Ingress analyzer | Very practical |
| High | Node analyzer | Explains scheduling/evictions |
| High | StatefulSet analyzer | Important for production workloads |
| High | NetworkPolicy analyzer | Hard Kubernetes problem |
| Medium | HPA analyzer | Useful operational diagnosis |
| Medium | DNS analyzer | Good dependency reasoning |
| Medium | Event timeline | Excellent incident context |
| Medium | Change detection | High value but requires historical data |
| Medium | JSON/exit codes | Important for automation |
| Later | Helm integration | Useful contextual data |
| Later | Argo CD integration | Useful deployment correlation |
| Later | Git integration | Change correlation |
| Later | LLM layer | Explanation/interface layer |
| Later | CRD-specific analyzers | Very broad surface area |

---

# 27. Overall Product Direction

The strongest evolution of KubeWhy is:

```text
                    KubeWhy
                       │
                       ▼
              Find unhealthy resources
                       │
                       ▼
              Build dependency graph
                       │
                       ▼
              Collect evidence
                       │
                       ▼
              Correlate failures
                       │
                       ▼
                Find root cause
                       │
             ┌─────────┴─────────┐
             ▼                   ▼
          Impact              Evidence
             │                   │
             └─────────┬─────────┘
                       ▼
                 Explain "why"
```

The product should ultimately answer:

> **"Something is broken. Show me the actual root cause, prove it with Kubernetes evidence, and tell me what is affected."**

That is a substantially broader and more differentiated product than a Pod troubleshooting utility, while still building directly on the current KubeWhy architecture.
