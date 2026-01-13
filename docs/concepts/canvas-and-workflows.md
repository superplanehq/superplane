# Canvas and Workflows

## Table of Contents

- [Overview](#overview)
- [Organizations](#organizations)
- [Canvas](#canvas)
- [Workflows](#workflows)
- [Cross-Workflow Coordination](#cross-workflow-coordination)

---

## Overview

SuperPlane organizes DevOps automation through a hierarchical structure: Organizations contain Canvases, and Canvases contain multiple Workflows.

```mermaid
graph TB
    O[Organization]
    O --> C1[Canvas-1]
    O --> C2[Canvas-2]
    
    C1 --> W1[Workflow-1]
    C1 --> W2[Workflow-2]
    
    C2 --> W3[Workflow-3]
```

---

## Organizations

Organizations provide the top-level boundary for all SuperPlane resources, operating as isolated tenants with complete data separation.

### Organization Structure Example

```mermaid
graph TB
    O[My-Company-Org] --> C1[Production-Canvas]
    O --> C2[Development-Canvas]
    
    C1 --> W1[CI-CD-Workflow]
    C1 --> W2[Incident-Response]
    
    C2 --> E1[GitHub-Source]
    E1 --> S1[Build-Stage]
    S1 --> S2[Test-Stage]
    S2 --> S3[Package-Stage]
    
    I1[GitHub-Integration]
    I2[Semaphore-Integration]
    
    C1 -.-> I1
    C1 -.-> I2
    C2 -.-> I1
    C2 -.-> I2
```

This example shows how integrations can be shared across Canvases while workflows remain isolated within their respective Canvas boundaries.

---

## Canvas

Canvas is your main workspace for building and managing DevOps workflows. Each Canvas operates as a self-contained project environment.

![Canvas Sidebar View](../images/sidebar.png)

### Canvas Management

Use Canvases to organize workflows by:
- **Application** - One Canvas per application or service
- **Environment** - Separate Canvases for dev/staging/production
- **Team** - Different Canvases for different team responsibilities

---

## Workflows

Workflows represent complete operational processes built from connected components.

### Workflow Patterns

**Linear chains:** Sequential component execution.

```mermaid
graph LR
    A[Repository-Push] --> B[Build] --> C[Test] --> D[Deploy] --> E[Notify]
```

**Parallel branches:** Multiple components execute simultaneously.

```mermaid
graph TB
    A[Repository-Push] --> B[Build]
    B --> C[Security-Scan]
    B --> D[Performance-Test]
    B --> E[Documentation-Update]
```

**Conditional routing:** Path selection based on conditions.

```mermaid
graph TB
    A[Build-Complete] --> B[Check-Environment]
    B -->|staging| C[Deploy-to-Staging]
    B -->|main| D[Manual-Approval]
    D --> E[Deploy-to-Production]
```

**Fan-out/fan-in:** Multiple parallel operations that converge.

```mermaid
graph TB
    A[Deploy-Request] --> B[Validation]
    B --> C[Security-Check]
    B --> D[Resource-Check]
    B --> E[Approval-Check]
    C --> F[Deploy-Decision]
    D --> F
    E --> F
```

### Workflow Example

```mermaid
graph LR
    E[GitHub-Source] --> S1[Build-Stage]
    S1 --> S2[Test-Stage]
    S2 --> S3[Deploy-Stage]
```

![Simple Super Plane Workflow](../images/core1.png)

---

## Cross-Workflow Coordination

Workflows can coordinate through several mechanisms:

### Orchestration Patterns

```mermaid
graph TB
    A1[Pipeline-Workflow-1] --> A2[Pipeline-Workflow-2] --> A3[Pipeline-Workflow-3]
    
    B1[Trigger-Event] --> B2[Parallel-Workflow-A]
    B1 --> B3[Parallel-Workflow-B]
    B1 --> B4[Parallel-Workflow-C]
    
    C1[Conditional-Event] --> C2{Check-Condition}
    C2 -->|Yes| C3[Workflow-X]
    C2 -->|No| C4[Workflow-Y]
```

**Pattern types:**
- **Pipeline workflows** - Sequential workflows where one triggers the next
- **Parallel workflows** - Independent workflows running simultaneously
- **Conditional workflows** - Different workflows triggered based on conditions