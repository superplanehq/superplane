# SuperPlane

SuperPlane ist eine **Open-Source DevOps-Steuerungsebene** zur Definition und Ausführung
ereignisbasierter Workflows. Es funktioniert mit den Tools, die Sie bereits verwenden, wie
Git, CI/CD, Observability, Incident Response, Infrastruktur und Benachrichtigungen.

![SuperPlane Screenshot](./screenshot.png)

## Projektstatus

<p>
  <a href="https://superplanehq.semaphoreci.com/projects/superplane"><img src="https://superplanehq.semaphoreci.com/badges/superplane/branches/main.svg?style=shields" alt="CI-Status auf Semaphore" /></a>
  <a href="https://github.com/superplanehq/superplane/pulse"><img src="https://img.shields.io/github/commit-activity/m/superplanehq/superplane" alt="GitHub Commit-Aktivität"/></a>
  <a href="https://discord.gg/KC78eCNsnw"><img src="https://img.shields.io/discord/1409914582239023200?label=discord" alt="Discord-Server" /></a>
</p>

Dieses Projekt befindet sich im Alpha-Stadium und entwickelt sich schnell weiter. Erwarten Sie raue Kanten und gelegentliche
Breaking Changes, während wir das Kernmodell und die Integrationen stabilisieren.
Wenn Sie es ausprobieren und auf etwas Verwirrendes stoßen, [erstellen Sie bitte ein Issue](https://github.com/superplanehq/superplane/issues/new).
Frühes Feedback ist äußerst wertvoll.

## Was es macht

- **Workflow-Orchestrierung**: Modellieren Sie mehrstufige operationelle Workflows, die mehrere Systeme umfassen.
- **Ereignisgesteuerte Automatisierung**: Lösen Sie Workflows durch Pushes, Deploy-Events, Alerts, Zeitpläne und Webhooks aus.
- **Steuerungsebenen-UI**: Entwerfen und verwalten Sie DevOps-Prozesse; inspizieren Sie Ausführungen, Status und Verlauf an einem Ort.
- **Gemeinsamer operationeller Kontext**: Halten Sie Workflow-Definitionen und operationelle Absichten in einem System, anstatt in verstreuten Skripten.

## Wie es funktioniert

- **Canvases**: Sie modellieren einen Workflow als gerichteten Graphen (ein "Canvas") aus Schritten und Abhängigkeiten.
- **Komponenten**: Jeder Schritt ist eine wiederverwendbare Komponente (eingebaut oder integrationsgestützt), die eine Aktion ausführt (z.B.: CI/CD aufrufen, einen Incident eröffnen, eine Benachrichtigung senden, auf eine Bedingung warten, Genehmigung erfordern).
- **Events & Trigger**: Eingehende Events (Webhooks, Zeitpläne, Tool-Events) werden mit Triggern abgeglichen und starten Ausführungen mit dem Event-Payload als Eingabe.
- **Ausführung + Sichtbarkeit**: SuperPlane führt den Graphen aus, verfolgt den Zustand und stellt Ausführungen/Verlauf/Debugging in der UI (und über die CLI) zur Verfügung.

### Beispielanwendungsfälle

Einige konkrete Dinge, die Teams mit SuperPlane erstellen:

- **Richtliniengesteuertes Produktions-Deployment**: Wenn CI grün abschließt, außerhalb der Geschäftszeiten zurückhalten, Bereitschafts- und Produktgenehmigung erfordern, dann das Deployment auslösen.
- **Progressive Delivery (10% → 30% → 60% → 100%)**: In Wellen deployen, bei jedem Schritt warten/verifizieren und bei Fehler mit einem Genehmigungsgate zurückrollen.
- **Release-Train mit Multi-Repo-Ship-Set**: Auf Tags/Builds von einer Reihe von Services warten, zusammenführen sobald alle bereit sind, dann ein koordiniertes Deployment auslösen.
- **"Erste 5 Minuten" Incident-Triage**: Bei Incident-Erstellung parallel Kontext abrufen (kürzliche Deployments + Gesundheitssignale), ein Beweispaket erstellen und ein Issue eröffnen.

## Schnellstart

Starten Sie den neuesten Demo-Container:

```
docker pull ghcr.io/superplanehq/superplane-demo:stable
docker run --rm -p 3000:3000 -v spdata:/app/data -ti ghcr.io/superplanehq/superplane-demo:stable
```

Öffnen Sie dann [http://localhost:3000](http://localhost:3000) und folgen Sie der [Schnellstartanleitung](https://docs.superplane.com/get-started/quickstart/).

## Unterstützte Integrationen

SuperPlane integriert sich mit den Tools, die Sie bereits verwenden. Jede Integration bietet Trigger (Events, die Workflows starten) und Komponenten (Aktionen, die Sie ausführen können).

> Die vollständige Liste finden Sie in unserer [Dokumentation](https://docs.superplane.com/components/). Fehlt ein Anbieter? [Erstellen Sie ein Issue](https://github.com/superplanehq/superplane/issues/new), um ihn anzufordern.

### KI & LLM

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/claude/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/claude.svg" alt="Claude"/><br/>Claude</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/openai/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/openai.svg" alt="OpenAI"/><br/>OpenAI</a></td>
</tr>
</table>

### Versionsverwaltung & CI/CD

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/github/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/github.svg" alt="GitHub"/><br/>GitHub</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/gitlab/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/gitlab.svg" alt="GitLab"/><br/>GitLab</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/semaphore/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/semaphore-logo-sign-black.svg" alt="Semaphore"/><br/>Semaphore</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/render/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/render.svg" alt="Render"/><br/>Render</a></td>
</tr>
</table>

### Cloud & Infrastruktur

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/aws/#ecr-•-on-image-push" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/aws.ecr.svg" alt="AWS ECR"/><br/>AWS ECR</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/aws/#lambda-•-run-function" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/aws.lambda.svg" alt="AWS Lambda"/><br/>AWS Lambda</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/aws/#code-artifact-•-on-package-version" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/aws.codeartifact.svg" alt="AWS CodeArtifact"/><br/>AWS CodeArtifact</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/cloudflare/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/cloudflare.svg" alt="Cloudflare"/><br/>Cloudflare</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/dockerhub/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/docker.svg" alt="DockerHub"/><br/>DockerHub</a></td>
</tr>
</table>

### Observability

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/datadog/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/datadog.svg" alt="DataDog"/><br/>DataDog</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/dash0/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/dash0.svg" alt="Dash0"/><br/>Dash0</a></td>
</tr>
</table>

### Incident-Management

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/pagerduty/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/pagerduty.svg" alt="PagerDuty"/><br/>PagerDuty</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/rootly/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/rootly.svg" alt="Rootly"/><br/>Rootly</a></td>
</tr>
</table>

### Kommunikation

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/discord/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/discord.svg" alt="Discord"/><br/>Discord</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/slack/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/slack.svg" alt="Slack"/><br/>Slack</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/sendgrid/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/sendgrid.svg" alt="SendGrid"/><br/>SendGrid</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/smtp/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/smtp.svg" alt="SMTP"/><br/>SMTP</a></td>
</tr>
</table>

### Ticketing

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/jira/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/jira.svg" alt="Jira"/><br/>Jira</a></td>
</tr>
</table>

### Entwicklerwerkzeuge

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/daytona/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/daytona.svg" alt="Daytona"/><br/>Daytona</a></td>
</tr>
</table>

## Produktionsinstallation

Sie können SuperPlane auf einem einzelnen Host oder auf Kubernetes bereitstellen:

- **[Einzelhost-Installation](https://docs.superplane.com/installation/overview/#single-host-installation)** - Bereitstellung auf AWS EC2, GCP Compute Engine oder anderen Cloud-Anbietern
- **[Kubernetes-Installation](https://docs.superplane.com/installation/overview/#kubernetes)** - Bereitstellung auf GKE, EKS oder jedem Kubernetes-Cluster

## Roadmap-Überblick

Dieser Abschnitt gibt einen schnellen Überblick darüber, was SuperPlane bereits unterstützt und was als Nächstes kommt.

**Bereits verfügbar**

✓ 75+ Komponenten  
✓ Ereignisgesteuerte Workflow-Engine  
✓ Visueller Canvas-Builder  
✓ Ausführungsverlauf, Event-Chain-Ansicht, Debug-Konsole  
✓ Starter-CLI und Beispiel-Workflows

**In Arbeit / geplant**

→ 200+ neue Komponenten (AWS, Grafana, DataDog, Azure, GitLab, Jira und mehr)  
→ [Canvas-Versionskontrolle](https://github.com/superplanehq/superplane/issues/1380)  
→ [SAML/SCIM](https://github.com/superplanehq/superplane/issues/1377) mit [erweitertem RBAC und Berechtigungen](https://github.com/superplanehq/superplane/issues/1378)  
→ [Artefakt-Versionsverfolgung](https://github.com/superplanehq/superplane/issues/1382)  
→ [Öffentliche API](https://github.com/superplanehq/superplane/issues/1854)

## Mitwirken

Wir freuen uns über Ihre Fehlerberichte, Verbesserungsvorschläge und gezielte PRs.

- Lesen Sie den **[Beitragsleitfaden](CONTRIBUTING.md)**, um loszulegen.
- Issues: Verwenden Sie GitHub Issues für Fehler und Feature-Anfragen.

## Lizenz

Apache License 2.0. Siehe `LICENSE`.

## Community

- **[Discord](https://discord.superplane.com)** - Treten Sie unserer Community bei für Diskussionen, Fragen und Zusammenarbeit
- **[X](https://x.com/superplanehq)** - Folgen Sie uns für Updates und Ankündigungen
