# Core Concepts

## Table of Contents

- [Overview](#overview)
- [Organizations](#organizations)
- [Canvas](#canvas)
- [Workflows](#workflows)
- [Components](#components)
- [Events and Data](#events-and-data)
- [Integrations](#integrations)
- [Security](#security)

---

## Overview

SuperPlane organizes DevOps automation around several key concepts that work together to create comprehensive workflow orchestration.

**The hierarchy:**
- **Organizations** provide data isolation and user management
- **Canvas** serves as your workspace for building workflows  
- **Workflows** are chains of connected components that automate processes
- **Components** are the building blocks that listen for events and perform actions
- **Events** carry data between components and trigger workflow execution

---

## Organizations

Organizations provide the top-level boundary for all SuperPlane resources, operating as isolated tenants with complete data separation.

**Key characteristics:**
- **Data isolation** - Complete separation between organizations with no shared resources
- **User management** - Organization admins control member access and permissions
- **Resource container** - All Canvases, integrations, and workflows exist within an organization

Organizations typically represent companies, business units, or major project divisions. Choose your organization scope based on data access requirements and administrative boundaries.

---

## Canvas

Canvas is your main workspace for building and managing DevOps workflows. Each Canvas operates as a self-contained project environment where you connect tools, build automation, and monitor execution.

**Primary functions:**
- **Tool integration** - Connect external services through integrations
- **Workflow construction** - Build multiple workflows using visual component connections
- **Real-time monitoring** - Observe workflow status, component states, and active executions
- **Component inspection** - View execution history, event logs, and configuration details

**Canvas workspace includes:**
- **Integration management** - Configure connections to external DevOps tools
- **Workflow builder** - Visual interface for connecting components into processes  
- **Execution dashboard** - Real-time view of workflow status and running processes
- **Secrets management** - Encrypted storage for API keys, tokens, and sensitive data
- **User permissions** - Canvas-level access control for team collaboration

![Canvas Sidebar View](../images/sidebar.png)

Canvas provides execution isolation - workflows, data, and state are contained within each Canvas boundary.

---

## Workflows

Workflows represent complete operational processes built from connected components. Multiple workflows can exist within a single Canvas, each handling different aspects of your DevOps requirements.

**Workflow characteristics:**
- **Event-driven progression** - Workflows advance through component-to-component event passing
- **Independent execution** - Workflows operate independently while sharing Canvas resources
- **Cross-tool orchestration** - Single workflows coordinate operations across multiple platforms

**Common workflow patterns:**
- **Linear chains** - Sequential component execution
- **Parallel branches** - Multiple components execute simultaneously  
- **Conditional routing** - Workflow path selection based on data or results

→ **Learn more:** [Canvas and Workflows](../concepts/canvas-and-workflows.md)

---

## Components

Components are the building blocks of workflows that operate on an event-driven model. They listen for events, process them, and emit new events to trigger downstream components.

**Component types:**
- **Event sources** - Listen to external systems and emit events to the Canvas
- **Stages** - Listen to events, perform actions on external resources, emit completion events  
- **Connection groups** - Coordinate events from multiple sources before emitting

**Key capabilities:**
- **Event filtering** - Components can filter incoming events using configurable rules
- **Input mapping** - Extract values from event payloads and map them to component inputs
- **Execution queues** - Manage FIFO queues of events awaiting processing
- **Condition control** - Apply approval requirements, time restrictions, and execution controls

**Executors** perform the actual work within stage components:
- **HTTP executors** - Make REST API calls to external services
- **Integration executors** - Trigger operations on connected platforms (GitHub Actions, Semaphore pipelines)

→ **Learn more:** [Components](../concepts/components.md)

---

## Events and Data

Events are the core mechanism driving workflow execution. Every component interaction, data transfer, and workflow progression happens through events.

**Event structure:**
- **Payload** - JSON data containing information passed between components
- **Headers** - Metadata about event source, timing, and routing information
- **Context** - SuperPlane-specific execution and tracking data

**Data flow:**
- **Inputs** define what data components expect to receive from events
- **Outputs** define what data components produce after execution  
- **Input mappings** extract and transform event data into component inputs
- **Executor parameters** explicitly pass input values to executors for use during execution
- **JSONPath expressions** parse event payloads to extract specific values

**Data sources:**
- External webhook payloads from integrated tools
- Component completion events containing execution results
- Previous component outputs forwarded as downstream inputs

→ **Learn more:** [Events and Data Flow](../concepts/events-and-data.md)

---

## Integrations

Integrations connect external DevOps tools to your Canvas, enabling two-way communication between SuperPlane and external systems.

**What integrations provide:**
- **API connectivity** - Authenticated connections to external tool APIs
- **Webhook registration** - Configure external tools to send events to SuperPlane
- **Credential management** - Secure storage of API tokens and authentication details
- **Component enablement** - Unlock tool-specific executors and event sources

**Supported integrations:**
- **GitHub** - Trigger Actions workflows, receive repository events
- **Semaphore** - Trigger CI/CD pipelines, receive build notifications
- **Custom webhooks** - Receive events from any system that can send HTTP requests

**Integration lifecycle:**
1. Configure authentication with target tool
2. Verify API permissions and connectivity
3. Set up webhook endpoints for event delivery
4. Use integration in components and executors

→ **Learn more:** [Integrations and Security](../concepts/integrations-security.md)

---

## Security

SuperPlane provides robust security for managing sensitive data and controlling access to workflows and resources.

**Secrets management:**
- **Canvas-scoped storage** - Encrypted secrets isolated per Canvas
- **Runtime injection** - Secure delivery of credentials to component execution environments
- **Zero-knowledge access** - Secret values encrypted and only decrypted during execution

**Access control:**
- **Organization-level isolation** - Complete data separation between organizations
- **Canvas permissions** - Control who can view, modify, and execute workflows
- **Role-based access** - Granular permissions for different user types

**Compliance features:**
- **Complete audit trails** - Every action, execution, and access logged with timestamps
- **Encrypted storage** - All sensitive data encrypted at rest and in transit
- **Secure authentication** - Integration with external identity providers

→ **Learn more:** [Integrations and Security](../concepts/integrations-security.md)