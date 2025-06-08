# ioshelfer：I/O 亚健康状态守护

[English Version](./README.md)

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![GitHub Stars](https://img.shields.io/github/stars/turtacn/ioshelfer.svg?style=social&label=Star)](https://github.com/turtacn/ioshelfer)

**`ioshelfer` 是一个由 eBPF 驱动、具备混沌工程能力的智能化 I/O 亚健康检测与自愈系统。它致力于在存储和网络I/O的“隐形杀手”级性能问题升级为严重故障前，主动识别并消除它们。**

---

### 核心痛点：隐形的 I/O 亚健康

在复杂的生产环境中，系统经常遭受 I/O “亚健康”状态的困扰：
- RAID 卡固件的 Bug 导致间歇性的高延迟。
- 老化的 SSD 性能缓慢下降，拖慢整个应用。
- 网络交换机端口出现微突发，导致 TCP 重传率飙升。

这些问题通常是瞬时的，传统监控（主要关注完全宕机）难以捕捉，但它们会悄无声息地侵蚀应用的 SLO，并最终导致雪崩式的故障。

### ioshelfer 的核心价值

`ioshelfer` 将运维模式从“被动救火”转变为“主动、自动化的系统健康管理”。

1.  **早期精准检测**: 利用 **eBPF** 技术实现内核级、低开销的监控，实时捕捉微小的性能异常（如 I/O 延迟抖动、队列深度问题）。
2.  **AI 驱动的预测**: 采用机器学习模型（如 LSTM），提前长达48小时预测硬盘、网卡等硬件的退化趋势。
3.  **自动化自愈**: 实现基于策略的智能修复，例如自动隔离不健康的磁盘、重路由网络流量，甚至自动回滚有问题的固件。
4.  **原生集成混沌工程**: 与 Litmus 等混沌工程框架无缝集成，通过主动注入 I/O 故障，持续验证系统的韧性。

### 主要功能特性

-   **多层次 I/O 监控**:
    -   **RAID 卡**: 在固件和驱动层追踪队列深度、IO 延迟和非致命错误率。
    -   **硬盘 (HDD/SSD)**: 结合 S.M.A.R.T. 属性分析与实时 I/O 性能指标（IOPS 波动、延迟标准差）。
    -   **网络 I/O**: 使用 eBPF 提供对 TCP/UDP 延迟、重传/丢包率和吞吐量下降的内核级洞察，支持 FC、iSCSI、RoCE 等多种协议。

-   **智能分析与预测**:
    -   强大的规则引擎，用于定义多维度的亚健康判定标准。
    -   可插拔的 AI/ML 引擎，基于历史时序数据进行预测性故障分析。

-   **策略驱动的自动修复**:
    -   **智能隔离**: 在保障业务连续性的前提下（例如，至少保留50%的可用路径），临时或永久隔离故障组件。
    -   **自愈脚本**: 触发自动化恢复工作流（如 RAID 卡重置、固件回滚、路径故障转移）。

-   **云原生与易于集成**:
    -   轻量级的 Agent 架构，可通过 Kubernetes Operator 进行部署。
    -   以 **Prometheus** 格式暴露指标，并支持 **OpenTelemetry**，与您现有的可观测性技术栈无缝集成。
    -   提供 Webhook 和 gRPC 接口，用于事件通知和 CMDB 同步。

-   **内置混沌工程能力**:
    -   可直接定义和执行 I/O 故障注入实验（如增加延迟、模拟丢包），以验证修复策略的有效性。

### 架构概览

`ioshelfer` 采用分层、模块化的架构，由节点级的 **Agent (代理)** 和一个中心的 **Control Plane (控制平面)** 组成。

-   **Agent**: 部署在每个被监控的节点上（在 K8s 中作为 DaemonSet 或作为普通系统服务）。它使用 eBPF 探针和其他收集器来采集数据，执行初步分析，并能执行本地的修复动作。
-   **Control Plane**: 聚合来自所有 Agent 的数据，运行复杂的分析和预测模型，管理策略，编排跨节点的修复任务，并为用户和外部系统提供 API。

详细设计请参见 [架构设计文档](./docs/architecture.md)。

![架构图](https://raw.githubusercontent.com/turtacn/ioshelfer/main/docs/images/architecture_overview_zh.png)
*(注：此图为 `docs/architecture.md` 中详细 Mermaid 图的简化版本)*

### 快速上手与用法示例

*（此部分将在代码库实现后，更新为实际的构建和运行命令。）*

#### 1. 运行 Agent (示例)

```bash
# 在主机上运行 agent 程序，并指向控制平面地址
./ioshelfer-agent --config ./agent.yaml --control-plane-addr=10.0.0.5:9090
````

#### 2\. 定义检测策略 (policy.yaml)

用户通过定义一个 YAML 文件来指定要检测的亚健康状况以及如何响应。

```yaml
apiVersion: "[ioshelfer.turtacn.com/v1alpha1](https://ioshelfer.turtacn.com/v1alpha1)"
kind: "SubHealthPolicy"
metadata:
  name: "critical-disk-latency"
spec:
  selector:
    # 应用于所有带有 'database' 标签的节点
    nodeLabel: "database"
  target:
    type: "Disk"
    # 目标为所有 NVMe 磁盘
    device: "/dev/nvme*"
  rules:
    # 如果95分位延迟连续5分钟超过20ms
    # 并且 IOPS 相对于基线下降超过30%，则触发
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
    # 规则触发时要执行的动作
    - name: "TemporaryIsolate"
      type: "script"
      # 限制设备的IO，并通知待命工程师
      script: "iotool --limit-iops /dev/nvme0n1 100 && notify-pagerduty --key 'CRITICAL_DISK_LATENCY' --details '设备 /dev/nvme0n1 处于亚健康状态'"
    - name: "PermanentIsolate"
      type: "api"
      # 在持续亚健康1小时后，触发数据重建
      triggerAfter: "1h"
      apiCall:
        service: "storage.k8s.io"
        action: "evictAndRebuild"
        params:
          device: "{TARGET_DEVICE}"
```

### 贡献代码

我们欢迎任何形式的贡献！无论是报告 Bug、提交功能请求、改进文档还是编写代码，我们都非常感谢您的帮助。

请阅读我们的 [贡献指南](/CONTRIBUTING.md) 来开始。

### 开源许可

`ioshelfer` 基于 Apache 2.0 许可开源。