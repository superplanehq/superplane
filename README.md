# SuperPlane

SuperPlane er et **åpen kildekode DevOps-kontrollplan** for å definere og kjøre
hendelsesbaserte arbeidsflyter. Det fungerer på tvers av verktøyene du allerede bruker, som
Git, CI/CD, observabilitet, hendelseshåndtering, infrastruktur og varslinger.

![SuperPlane screenshot](./screenshot.png)

## Prosjektstatus

<p>
  <a href="https://superplanehq.semaphoreci.com/projects/superplane"><img src="https://superplanehq.semaphoreci.com/badges/superplane/branches/main.svg?style=shields" alt="CI Status on Semaphore" /></a>
  <a href="https://github.com/superplanehq/superplane/pulse"><img src="https://img.shields.io/github/commit-activity/m/superplanehq/superplane" alt="GitHub commit activity"/></a>
  <a href="https://discord.gg/KC78eCNsnw"><img src="https://img.shields.io/discord/1409914582239023200?label=discord" alt="Discord server" /></a>
</p>

Dette prosjektet er i alfa og utvikler seg raskt. Forvent uferdige kanter og av og til
endringer som bryter kompatibilitet mens vi stabiliserer kjernemodellen og integrasjonene.
Hvis du prøver det og støter på noe forvirrende, kan du gjerne [opprette en issue](https://github.com/superplanehq/superplane/issues/new).
Tidlig tilbakemelding er svært verdifull.

## Hva det gjør

- **Orkestrering av arbeidsflyt**: Modellér flertrinns operasjonelle arbeidsflyter som går på tvers av flere systemer.
- **Hendelsesdrevet automasjon**: Start arbeidsflyter fra push, deploy-hendelser, varsler, tidsplaner og webhooks.
- **Kontrollplan-UI**: Design og administrer DevOps-prosesser; inspiser kjøringer, status og historikk på ett sted.
- **Felles operasjonell kontekst**: Hold definisjoner av arbeidsflyter og operasjonell intensjon i ett system i stedet for spredte script.

## Slik fungerer det

- **Canvas**: Du modellerer en arbeidsflyt som en rettet graf (en “Canvas”) av steg og avhengigheter.
- **Komponenter**: Hvert steg er en gjenbrukbar komponent (innebygd eller via integrasjon) som utfører en handling (for eksempel: kalle CI/CD, opprette en hendelse, poste et varsel, vente på en tilstand, kreve godkjenning).
- **Hendelser og triggere**: Innkommende hendelser (webhooks, tidsplaner, verktøy-hendelser) matcher triggere og starter kjøringer med hendelsesdata som input.
- **Kjøring + innsyn**: SuperPlane kjører grafen, sporer tilstand, og viser kjøringer/historikk/feilsøking i UI (og via CLI).

### Eksempel på brukstilfeller

Noen konkrete ting team bygger med SuperPlane:

- **Produksjonsdeploy med policy-gate**: når CI er grønn, hold utenfor arbeidstid, krev on-call + produktgodkjenning, og trigge deploy.
- **Progressiv utrulling (10% → 30% → 60% → 100%)**: deploy i bølger, vent/verifiser i hvert steg, og rull tilbake ved feil med en godkjennings-gate.
- **Release train med multi-repo ship set**: vent på tags/builds fra et sett med tjenester, samle når alle er klare, og start en koordinert deploy.
- **“Første 5 minutter” incident-triage**: når en hendelse opprettes, hent kontekst parallelt (siste deploys + helsesignaler), generer en evidenspakke og opprett en issue.

## Hurtigstart

Kjør den nyeste demo-containeren:

```
docker pull ghcr.io/superplanehq/superplane-demo:stable
docker run --rm -p 3000:3000 -v spdata:/app/data -ti ghcr.io/superplanehq/superplane-demo:stable
```

Åpne deretter [http://localhost:3000](http://localhost:3000) og følg [hurtigstartguiden](https://docs.superplane.com/get-started/quickstart/).

## Støttede integrasjoner

SuperPlane integrerer med verktøyene du allerede bruker. Hver integrasjon gir triggere (hendelser som starter arbeidsflyter) og komponenter (handlinger du kan kjøre).

> Se hele listen i [dokumentasjonen](https://docs.superplane.com/components/). Mangler det en leverandør? [Opprett en issue](https://github.com/superplanehq/superplane/issues/new) for å be om den.

### AI & LLM

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/claude/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/claude.svg" alt="Claude"/><br/>Claude</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/openai/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/openai.svg" alt="OpenAI"/><br/>OpenAI</a></td>
</tr>
</table>

### Versjonskontroll og CI/CD

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/github/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/github.svg" alt="GitHub"/><br/>GitHub</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/gitlab/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/gitlab.svg" alt="GitLab"/><br/>GitLab</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/semaphore/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/semaphore-logo-sign-black.svg" alt="Semaphore"/><br/>Semaphore</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/render/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/render.svg" alt="Render"/><br/>Render</a></td>
</tr>
</table>

### Sky og infrastruktur

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/aws/#ecr-•-on-image-push" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/aws.ecr.svg" alt="AWS ECR"/><br/>AWS ECR</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/aws/#lambda-•-run-function" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/aws.lambda.svg" alt="AWS Lambda"/><br/>AWS Lambda</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/aws/#code-artifact-•-on-package-version" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/aws.codeartifact.svg" alt="AWS CodeArtifact"/><br/>AWS CodeArtifact</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/cloudflare/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/cloudflare.svg" alt="Cloudflare"/><br/>Cloudflare</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/dockerhub/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/docker.svg" alt="DockerHub"/><br/>DockerHub</a></td>
</tr>
</table>

### Observabilitet

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/datadog/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/datadog.svg" alt="DataDog"/><br/>DataDog</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/dash0/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/dash0.svg" alt="Dash0"/><br/>Dash0</a></td>
</tr>
</table>

### Hendelseshåndtering

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/pagerduty/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/pagerduty.svg" alt="PagerDuty"/><br/>PagerDuty</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/rootly/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/rootly.svg" alt="Rootly"/><br/>Rootly</a></td>
</tr>
</table>

### Kommunikasjon

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/discord/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/discord.svg" alt="Discord"/><br/>Discord</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/slack/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/slack.svg" alt="Slack"/><br/>Slack</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/sendgrid/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/sendgrid.svg" alt="SendGrid"/><br/>SendGrid</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/smtp/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/smtp.svg" alt="SMTP"/><br/>SMTP</a></td>
</tr>
</table>

### Saker / ticketing

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/jira/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/jira.svg" alt="Jira"/><br/>Jira</a></td>
</tr>
</table>

### Utviklerverktøy

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/daytona/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/daytona.svg" alt="Daytona"/><br/>Daytona</a></td>
</tr>
</table>

## Installasjon i produksjon

Du kan kjøre SuperPlane på en enkelt host eller på Kubernetes:

- **[Installasjon på én host](https://docs.superplane.com/installation/overview/#single-host-installation)** - Deploy på AWS EC2, GCP Compute Engine eller andre skyleverandører
- **[Kubernetes-installasjon](https://docs.superplane.com/installation/overview/#kubernetes)** - Deploy på GKE, EKS eller hvilken som helst Kubernetes-klynge

## Oversikt over veikart

Denne seksjonen gir et raskt overblikk over hva SuperPlane støtter nå og hva som kommer.

**Tilgjengelig nå**

✓ 75+ components  
✓ Hendelsesdrevet arbeidsflytmotor  
✓ Visuell Canvas-bygger  
✓ Kjøringshistorikk, event chain-visning, debug-konsoll  
✓ Enkel CLI og eksempelarbeidsflyter

**Under arbeid / kommer**

→ 200+ nye komponenter (AWS, Grafana, DataDog, Azure, GitLab, Jira og mer)  
→ [Canvas version control](https://github.com/superplanehq/superplane/issues/1380)  
→ [SAML/SCIM](https://github.com/superplanehq/superplane/issues/1377) med [utvidet RBAC og rettigheter](https://github.com/superplanehq/superplane/issues/1378)  
→ [Versjonssporing av artefakter](https://github.com/superplanehq/superplane/issues/1382)  
→ [Offentlig API](https://github.com/superplanehq/superplane/issues/1854)

## Bidra

Vi ønsker velkommen feilrapporter, forbedringsforslag og fokuserte PR-er.

- Les **[Contributing Guide](CONTRIBUTING.md)** for å komme i gang.
- Issues: bruk GitHub issues for feil og feature-forespørsler.

## Lisens

Apache License 2.0. Se `LICENSE`.

## Fellesskap

- **[Discord](https://discord.superplane.com)** - Bli med for diskusjoner, spørsmål og samarbeid
- **[X](https://x.com/superplanehq)** - Følg oss for oppdateringer og annonseringer
