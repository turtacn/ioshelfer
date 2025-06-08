# ioshelfer: The I/O Sub-Health Guardian

[中文版](./README-zh.md)

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![GitHub Stars](https://img.shields.io/github/stars/turtacn/ioshelfer.svg?style=social&label=Star)](https://github.com/turtacn/ioshelfer)

**`ioshelfer` is an intelligent, eBPF-powered, and chaos-ready I/O sub-health detection and self-healing system. It proactively identifies and mitigates "silent killer" performance issues in storage and networking before they escalate into critical failures.**

---

### The Pain Point: Silent I/O Sub-Health

In complex production environments, systems often suffer from I/O "sub-health" conditions:
- A RAID controller's firmware bug causes intermittent high latency.
- An aging SSD's performance degrades slowly, causing application slowdowns.
- A network switch port experiences micro-bursts, leading to high TCP retransmission rates.

These issues are often transient, hard to detect with traditional monitoring (which focuses on complete failures), and can silently cripple application SLOs, leading to cascading failures.

### The Core Value of ioshelfer

`ioshelfer` shifts the paradigm from reactive firefighting to proactive, automated system wellness management.

1.  **Early & Accurate Detection**: Leverages **eBPF** for kernel-level, low-overhead monitoring to catch subtle anomalies (e.g., I/O latency spikes, queue depth issues) in real-time.
2.  **AI-Powered Prediction**: Utilizes ML models (e.g., LSTM) to predict hardware degradation trends (disks, NICs) up to 48 hours in advance.
3.  **Automated Self-Healing**: Implements intelligent, policy-based remediation, such as isolating unhealthy disks, rerouting network traffic, or automatically rolling back faulty firmware.
4.  **Chaos Engineering Ready**: Integrates seamlessly with chaos engineering frameworks like Litmus to inject I/O faults, continuously validating the system's resilience.

### Key Features

-   **Multi-Layer I/O Monitoring**:
    -   **RAID Cards**: Tracks queue depth, I/O latency, and non-fatal error rates at both firmware and driver levels.
    -   **Disks (HDD/SSD)**: Combines S.M.A.R.T. attribute analysis with real-time I/O performance metrics (IOPS fluctuations, latency standard deviation).
    -   **Network I/O**: Provides kernel-level insights into TCP/UDP latency, retransmission/loss rates, and throughput degradation using eBPF, supporting protocols like FC, iSCSI, and RoCE.

-   **Intelligent Analysis & Prediction**:
    -   A powerful rules engine for defining multi-dimensional sub-health criteria.
    -   A pluggable AI/ML engine for predictive failure analysis based on historical time-series data.

-   **Policy-Driven Automated Remediation**:
    -   **Smart Isolation**: Temporarily or permanently isolates faulty components while ensuring service continuity (e.g., maintaining at least 50% of available paths).
    -   **Self-Healing Scripts**: Triggers automated recovery workflows (e.g., RAID controller reset, firmware rollback, path failover).

-   **Cloud-Native & Integration-Friendly**:
    -   Lightweight, agent-based architecture deployable via a Kubernetes Operator.
    -   Exposes metrics in **Prometheus** format and supports **OpenTelemetry** for seamless integration with existing observability stacks.
    -   Provides Webhook and gRPC interfaces for event notifications and CMDB synchronization.

-   **Built-in Chaos Engineering**:
    -   Define and execute I/O fault injection experiments (e.g., high latency, packet loss) directly to validate remediation policies.

### Architecture Overview

`ioshelfer` adopts a layered, modular architecture, consisting of node-level **Agents** and a central **Control Plane**.

-   **Agent**: Deployed on each monitored node (as a DaemonSet in K8s or a system service). It uses eBPF probes and other collectors to gather data, performs initial analysis, and can execute local remediation actions.
-   **Control Plane**: Aggregates data from agents, runs complex analysis and prediction models, manages policies, orchestrates cross-node remediation, and exposes APIs for users and external systems.

For a detailed design, please see the [Architecture Document](./docs/architecture.md).

![Architecture Diagram](https://raw.githubusercontent.com/turtacn/ioshelfer/main/docs/images/architecture_overview_en.png)
*(Note: This image would be a simplified version of the detailed Mermaid diagram in `docs/architecture.md`)*

### Quick Start & Usage Snippets

*(This section will be updated with actual build/run commands once the codebase is implemented.)*

#### 1. Running the Agent (Example)

```bash
# Run the agent binary on a host, pointing to the control plane
./ioshelfer-agent --config ./agent.yaml --control-plane-addr=10.0.0.5:9090
````

#### 2\. Defining a Detection Policy (policy.yaml)

A user defines a YAML file to specify what sub-health condition to detect and how to react.

```yaml
apiVersion: "[ioshelfer.turtacn.com/v1alpha1](https://ioshelfer.turtacn.com/v1alpha1)"
kind: "SubHealthPolicy"
metadata:
  name: "critical-disk-latency"
spec:
  selector:
    # Apply to all nodes with the 'database' label
    nodeLabel: "database"
  target:
    type: "Disk"
    # Target all NVMe disks
    device: "/dev/nvme*"
  rules:
    # Trigger if 95th percentile latency is over 20ms for 5 minutes
    # AND IOPS drop by more than 30% from the baseline
    - name: "HighLatencyAndIOPSDrop"
      condition: "AND"
      metrics:
        - name: "p95_latency_ms"
          operator: "GreaterThan"
          value: 20
        - name: "iops_drop_percent"
          operator: "GreaterThan"
          value: 30
      duration: "5m"
  remediation:
    # Action to take when the rule is triggered
    - name: "TemporaryIsolate"
      type: "script"
      # Limits I/O to the device and notifies the on-call engineer
      script: "iotool --limit-iops /dev/nvme0n1 100 && notify-pagerduty --key 'CRITICAL_DISK_LATENCY' --details 'Device /dev/nvme0n1 is sub-healthy'"
    - name: "PermanentIsolate"
      type: "api"
      # Triggers a data rebuild after 1 hour of sustained sub-health
      triggerAfter: "1h"
      apiCall:
        service: "storage.k8s.io"
        action: "evictAndRebuild"
        params:
          device: "{TARGET_DEVICE}"
```

### Contributing

We welcome contributions of all kinds\! Whether it's reporting a bug, submitting a feature request, improving documentation, or writing code, we appreciate your help.

Please read our [Contributing Guidelines](./CONTRIBUTING.md) to get started.

### License

`ioshelfer` is licensed under the Apache 2.0 License.