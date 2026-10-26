# SuperPlane architecture and design system

These maps help new users and contributors see how SuperPlane fits together.
Use the linked guides for implementation details and planning changes.

## System architecture

[![SuperPlane system architecture](images/system-architecture.png)](images/system-architecture.png)

Select either diagram to view it at full resolution.

1. The web UI and CLI send HTTP requests to the Go application. External services send events through webhook routes.
2. The REST gateway calls application services in the same process. Services store application state and pending work in PostgreSQL.
3. Workers process events and execute workflow components. RabbitMQ notifications and database scans tell workers when work is available.
4. Components call external APIs or create tasks for runners, which execute shell commands and agent tools.
5. Results update stored state. The browser receives state notifications through WebSocket connections and reads logs through separate HTTP routes.

PostgreSQL stores durable work; RabbitMQ carries notifications.
Factory lines use the same canvas execution engine.
Fleet Manager reads capacity through the installation admin API and provisions runner hosts through cloud or Docker providers.
Application roles can run together or separately, depending on deployment configuration.

See the [architecture guide](../contributing/architecture.md) for execution, storage, permissions, and source directories.
See the [factory design](../design/factory.md) and [runner log storage design](../design/active-runner-task-log-storage.md) for those subsystems.

## Design system

[![SuperPlane design system map](images/design-system.png)](images/design-system.png)

The interface combines shared styles with several component families.
The arrows show composition and review relationships, not strict import boundaries.

| Area | Starting point |
| --- | --- |
| Colors and themes | [App.css](../../web_src/src/App.css) defines semantic tokens; [ThemeProvider](../../web_src/src/contexts/ThemeProvider.tsx) controls light and dark preferences. |
| Typography and icons | Reuse [global styles](../../web_src/src/index.css), [Icon](../../web_src/src/components/Icon), and the [icon registry](../../web_src/src/lib/iconRegistry.ts). |
| Shared controls | Prefer [shadcn controls](../../web_src/src/components/ui) for forms. Existing screens also use [ReUI](../../web_src/src/components/reui) and [ui](../../web_src/src/ui). |
| Product screens | [Factory pages](../../web_src/src/pages/factories) and [app pages](../../web_src/src/pages/app) compose controls. [Mappers](../../web_src/src/pages/app/mappers) adapt integration data for canvas views. |
| Visual review | [Storybook](../../web_src/.storybook) shows component states. The token catalog is **Factories → Design System → Tokens**. |

Follow the [frontend component guide](../../web_src/AGENTS.md) and reuse nearby patterns before adding controls.
Use the [component design guide](../contributing/component-design.md) for canvas components and integration output.
Stories support visual review; automated tests check behavior.

## Diagram scope

The images reflect [source revision bfb220601](https://github.com/superplanehq/superplane/commit/bfb220601dae150e1f635a840ac597c0d6ad669d).
The system map shows the integrated runner path.
It omits managed-agent sessions and the legacy execution path available at that revision.
Current runner tasks use the integrated Runner API; legacy broker access remains for historical logs.
