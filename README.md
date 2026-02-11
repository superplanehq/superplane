# SuperPlane

Το SuperPlane είναι ένα **open source DevOps control plane** για να ορίζεις και να εκτελείς
workflows που βασίζονται σε events. Δουλεύει πάνω στα εργαλεία που ήδη χρησιμοποιείς, όπως
Git, CI/CD, observability, incident response, υποδομή (infra) και ειδοποιήσεις.

![SuperPlane screenshot](./screenshot.png)

## Κατάσταση έργου

<p>
  <a href="https://superplanehq.semaphoreci.com/projects/superplane"><img src="https://superplanehq.semaphoreci.com/badges/superplane/branches/main.svg?style=shields" alt="CI Status on Semaphore" /></a>
  <a href="https://github.com/superplanehq/superplane/pulse"><img src="https://img.shields.io/github/commit-activity/m/superplanehq/superplane" alt="GitHub commit activity"/></a>
  <a href="https://discord.gg/KC78eCNsnw"><img src="https://img.shields.io/discord/1409914582239023200?label=discord" alt="Discord server" /></a>
</p>

Το project είναι σε alpha και εξελίσσεται γρήγορα. Περίμενε «γωνίες», και κατά διαστήματα
αλλαγές που μπορεί να σπάνε πράγματα, μέχρι να σταθεροποιηθεί το βασικό μοντέλο και οι integrations.
Αν το δοκιμάσεις και κάτι σε μπερδέψει, άνοιξε ένα issue: [open an issue](https://github.com/superplanehq/superplane/issues/new).
Το early feedback βοηθάει πολύ.

## Τι κάνει

- **Ορχήστρωση workflows**: Μοντελοποίηση πολυβηματικών operational workflows που απλώνονται σε πολλά συστήματα.
- **Αυτοματοποίηση με events**: Triggers από pushes, deploy events, alerts, schedules και webhooks.
- **UI control plane**: Σχεδίαση/διαχείριση DevOps διαδικασιών· προβολή runs, status και ιστορικού σε ένα σημείο.
- **Κοινό operational context**: Κρατάς ορισμούς workflows και «πρόθεση» λειτουργίας σε ένα σύστημα, αντί για διάσπαρτα scripts.

## Πώς δουλεύει

- **Canvases**: Μοντελοποιείς ένα workflow ως κατευθυνόμενο γράφο (ένα “Canvas”) με βήματα και εξαρτήσεις.
- **Components**: Κάθε βήμα είναι ένα επαναχρησιμοποιήσιμο component (built-in ή μέσω integration) που κάνει μια ενέργεια (π.χ. call CI/CD, άνοιγμα incident, notification, αναμονή συνθήκης, approval).
- **Events & triggers**: Εισερχόμενα events (webhooks, schedules, events από εργαλεία) ταιριάζουν με triggers και ξεκινούν εκτελέσεις με input το payload.
- **Εκτέλεση + ορατότητα**: Το SuperPlane εκτελεί τον γράφο, παρακολουθεί κατάσταση, και εμφανίζει runs/ιστορικό/debugging στο UI (και μέσω CLI).

### Παραδείγματα χρήσης

Μερικά συγκεκριμένα πράγματα που φτιάχνουν ομάδες με το SuperPlane:

- **Production deploy με policies/approvals**: όταν το CI γίνει green, αναμονή εκτός ωραρίου, approvals (on-call + product), και μετά deploy.
- **Progressive delivery (10% → 30% → 60% → 100%)**: deploy σε κύματα, wait/verify σε κάθε βήμα, και rollback σε failure με approval gate.
- **Release train με multi-repo ship set**: αναμονή για tags/builds από σύνολο services, fan-in όταν είναι όλα έτοιμα, και μετά συντονισμένο deploy.
- **Incident triage “πρώτα 5 λεπτά”**: όταν δημιουργηθεί incident, parallel fetch context (πρόσφατα deploys + health signals), evidence pack, και άνοιγμα issue.

## Γρήγορη εκκίνηση

Τρέξε το τελευταίο demo container:

```
docker pull ghcr.io/superplanehq/superplane-demo:stable
docker run --rm -p 3000:3000 -v spdata:/app/data -ti ghcr.io/superplanehq/superplane-demo:stable
```

Μετά άνοιξε το [http://localhost:3000](http://localhost:3000) και ακολούθησε το [quick start guide](https://docs.superplane.com/get-started/quickstart/).

## Υποστηριζόμενες ενσωματώσεις (Integrations)

Το SuperPlane συνδέεται με τα εργαλεία που ήδη χρησιμοποιείς. Κάθε integration παρέχει triggers (events που ξεκινούν workflows) και components (actions που μπορείς να τρέξεις).

> Δες την πλήρη λίστα στην [τεκμηρίωση](https://docs.superplane.com/components/). Λείπει κάποιος provider; άνοιξε issue: [Open an issue](https://github.com/superplanehq/superplane/issues/new).

### AI & LLM

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/claude/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/claude.svg" alt="Claude"/><br/>Claude</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/openai/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/openai.svg" alt="OpenAI"/><br/>OpenAI</a></td>
</tr>
</table>

### Version Control & CI/CD

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/github/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/github.svg" alt="GitHub"/><br/>GitHub</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/gitlab/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/gitlab.svg" alt="GitLab"/><br/>GitLab</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/semaphore/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/semaphore-logo-sign-black.svg" alt="Semaphore"/><br/>Semaphore</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/render/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/render.svg" alt="Render"/><br/>Render</a></td>
</tr>
</table>

### Cloud & Infrastructure

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

### Incident Management

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/pagerduty/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/pagerduty.svg" alt="PagerDuty"/><br/>PagerDuty</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/rootly/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/rootly.svg" alt="Rootly"/><br/>Rootly</a></td>
</tr>
</table>

### Communication

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

### Developer Tools

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/daytona/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/daytona.svg" alt="Daytona"/><br/>Daytona</a></td>
</tr>
</table>

## Production εγκατάσταση

Μπορείς να κάνεις deploy το SuperPlane σε single host ή σε Kubernetes:

- **[Single Host Installation](https://docs.superplane.com/installation/overview/#single-host-installation)** - Deploy σε AWS EC2, GCP Compute Engine ή άλλους cloud providers
- **[Kubernetes Installation](https://docs.superplane.com/installation/overview/#kubernetes)** - Deploy σε GKE, EKS ή οποιοδήποτε Kubernetes cluster

## Roadmap (σύνοψη)

Μια γρήγορη εικόνα για το τι υποστηρίζει ήδη το SuperPlane και τι έρχεται.

**Διαθέσιμα τώρα**

✓ 75+ components  
✓ Event-driven workflow engine  
✓ Visual Canvas builder  
✓ Run history, event chain view, debug console  
✓ Starter CLI και παραδείγματα workflows

**Σε εξέλιξη / έρχεται**

→ 200+ νέα components (AWS, Grafana, DataDog, Azure, GitLab, Jira, κ.ά.)  
→ [Canvas version control](https://github.com/superplanehq/superplane/issues/1380)  
→ [SAML/SCIM](https://github.com/superplanehq/superplane/issues/1377) με [extended RBAC και permissions](https://github.com/superplanehq/superplane/issues/1378)  
→ [Artifact version tracking](https://github.com/superplanehq/superplane/issues/1382)  
→ [Public API](https://github.com/superplanehq/superplane/issues/1854)

## Συνεισφορά (Contributing)

Καλωσορίζουμε bug reports, ιδέες για βελτιώσεις και στοχευμένα PRs.

- Διάβασε το **[Contributing Guide](CONTRIBUTING.md)** για να ξεκινήσεις.
- Για bugs/feature requests χρησιμοποίησε τα GitHub issues.

## Άδεια

Apache License 2.0. Δες το `LICENSE`.

## Κοινότητα

- **[Discord](https://discord.superplane.com)** - Έλα στην κοινότητα για συζητήσεις, ερωτήσεις και συνεργασία
- **[X](https://x.com/superplanehq)** - Ακολούθησέ μας για updates και ανακοινώσεις
