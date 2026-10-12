import type { FactoriesAutomationRef, FactoriesWorkOrder, FactoriesWorkOrderArtifact } from "@/api-client";
import bitbucketIcon from "@/assets/icons/integrations/bitbucket.svg";
import datadogIcon from "@/assets/icons/integrations/datadog.svg";
import dependabotIcon from "@/assets/icons/integrations/dependabot.svg";
import githubIcon from "@/assets/icons/integrations/github.svg";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import linearIcon from "@/assets/icons/integrations/linear.svg";
import pagerdutyIcon from "@/assets/icons/integrations/pagerduty.svg";
import productiveIcon from "@/assets/icons/integrations/productive.svg";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import slackIcon from "@/assets/icons/integrations/slack.svg";
import superplaneIcon from "@/assets/superplane.svg";
import { getUserInitials, type OrgUserDisplay, type OrgUserDisplayLookup } from "@/lib/orgUserDisplay";

import {
  STORYBOOK_ME_USER_AVATAR_URL,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
} from "../../__fixtures__/factoryPageResponses";
import { extractArtifactUrl, toArtifactDataRecord } from "../../lib/workOrderArtifact";
import { reviewCandidateForWorkOrderId } from "../onboarding/first-run/reviewCandidates";
import { canvasKeyForAutomation } from "./splitRunCanvases";

export const CREATED_MANUALLY = "Created manually";

export type SplitRunIntakeKind =
  | "github-issues"
  | "bitbucket-prs"
  | "dependabot-alerts"
  | "jira-issues"
  | "sentry-exceptions"
  | "pagerduty-incidents"
  | "productive-tasks"
  | "datadog"
  | "linear-issues"
  | "slack";

export type SplitRunAddedBy =
  | { kind: "intake"; name: string }
  | { kind: "manual" }
  | { kind: "imported"; personName: string }
  | { kind: "mcp"; name: string };

export type SplitRunSource =
  | {
      kind: "intake";
      name: string;
      iconSrc: string;
      iconAlt: string;
      ticket?: { label: string; href: string };
      addedBy?: SplitRunAddedBy;
    }
  | {
      kind: "mcp";
      name: string;
      iconSrc: string;
      iconAlt: string;
    }
  | {
      kind: "manual";
      person: OrgUserDisplay;
      detail: typeof CREATED_MANUALLY;
      addedBy?: SplitRunAddedBy;
    };

export function addedByForSource(source: SplitRunSource): SplitRunAddedBy {
  if (source.kind === "mcp") {
    return { kind: "mcp", name: source.name };
  }
  if (source.addedBy) {
    return source.addedBy;
  }
  if (source.kind === "manual") {
    return { kind: "manual" };
  }
  return { kind: "intake", name: source.name };
}

export function isMonochromeSourceLogo(iconAlt: string): boolean {
  return iconAlt === "GitHub" || iconAlt === "SuperPlane";
}

const SOURCE_PERSON_FALLBACK: OrgUserDisplay = {
  id: STORYBOOK_ME_USER_ID,
  name: STORYBOOK_ME_USER_NAME,
  initials: getUserInitials(STORYBOOK_ME_USER_NAME),
  avatarUrl: STORYBOOK_ME_USER_AVATAR_URL,
};

export const INTAKE_PRESENTATION: Record<SplitRunIntakeKind, { name: string; iconSrc: string; iconAlt: string }> = {
  "github-issues": { name: "GitHub issues", iconSrc: githubIcon, iconAlt: "GitHub" },
  "bitbucket-prs": { name: "Bitbucket pull requests", iconSrc: bitbucketIcon, iconAlt: "Bitbucket" },
  "dependabot-alerts": { name: "Dependabot alerts", iconSrc: dependabotIcon, iconAlt: "Dependabot" },
  "jira-issues": { name: "Jira issues", iconSrc: jiraIcon, iconAlt: "Jira" },
  "sentry-exceptions": { name: "Sentry exceptions", iconSrc: sentryIcon, iconAlt: "Sentry" },
  "pagerduty-incidents": { name: "PagerDuty incidents", iconSrc: pagerdutyIcon, iconAlt: "PagerDuty" },
  "productive-tasks": { name: "Productive tasks", iconSrc: productiveIcon, iconAlt: "Productive" },
  datadog: { name: "Datadog errors", iconSrc: datadogIcon, iconAlt: "Datadog" },
  "linear-issues": { name: "Linear", iconSrc: linearIcon, iconAlt: "Linear" },
  slack: { name: "Slack", iconSrc: slackIcon, iconAlt: "Slack" },
};

// Sources an intake app or a ticket link can be recognized by, before the
// GitHub default applies. GitHub issues need no hint: the intake app is named
// after its issues, and an issue link has no other marker. Dependabot alerts
// share github.com, so the alert path is classified first. This hint only
// names an automation when the task has no origin link.
const INTAKE_KIND_HINTS: Array<{ pattern: RegExp; kind: SplitRunIntakeKind }> = [
  { pattern: /dependabot/i, kind: "dependabot-alerts" },
  { pattern: /bitbucket/i, kind: "bitbucket-prs" },
  { pattern: /jira/i, kind: "jira-issues" },
  { pattern: /productive/i, kind: "productive-tasks" },
  { pattern: /pagerduty/i, kind: "pagerduty-incidents" },
  { pattern: /datadog|ddog-gov\.com/i, kind: "datadog" },
  { pattern: /linear/i, kind: "linear-issues" },
];

export function sourceTicketLabel(url: string): string {
  const parsed = parseUrl(url);
  if (!parsed) {
    return url;
  }
  const bitbucket = bitbucketPullRequestLabel(parsed);
  if (bitbucket) {
    return bitbucket;
  }
  const github = githubTicketLabel(parsed);
  if (github) {
    return github;
  }
  const jira = jiraTicketLabel(parsed);
  if (jira) {
    return jira;
  }
  const linear = linearTicketLabel(parsed);
  if (linear) {
    return linear;
  }
  const id = hostTicketId(parsed);
  // Productive.io links carry the organization id where other hosts carry a
  // name. A number says nothing to a reader, so the task id stands alone.
  if (parsed.hostname.endsWith("productive.io")) {
    return id ? `#${id}` : parsed.hostname;
  }
  const org = parsed.hostname.split(".")[0] ?? parsed.hostname;
  if (id) {
    return `${org}#${id}`;
  }
  return parsed.hostname;
}

export function splitRunIntakeSource(href: string, intakeKind?: SplitRunIntakeKind): SplitRunSource {
  return intakeSourceFromHref(href, intakeKind);
}

export function splitRunSourceForOrder(order: FactoriesWorkOrder, resolveUser?: OrgUserDisplayLookup): SplitRunSource {
  const originHref = order.origin?.url?.trim();
  if (originHref) {
    const source = intakeSourceFromHref(
      originHref,
      intakeKindFromHref(originHref),
      order.origin?.label?.trim() || sourceTicketLabel(originHref),
    );
    return { ...source, addedBy: addedByForIntakeOrder(order, source.name) };
  }

  const candidate = reviewCandidateForWorkOrderId(order.id);
  if (candidate?.issue.url) {
    const source = intakeSourceFromHref(candidate.issue.url);
    return { ...source, addedBy: addedByForIntakeOrder(order, source.name) };
  }

  const automation = order.createdBy?.automation;
  if (automation) {
    const source = intakeSourceFromKind(intakeKindForAutomation(automation));
    return { ...source, addedBy: { kind: "intake", name: source.name } };
  }

  const mcpName = order.mcpClient?.name?.trim();
  if (mcpName) {
    return {
      kind: "mcp",
      name: mcpName,
      iconSrc: superplaneIcon,
      iconAlt: "SuperPlane",
    };
  }

  return {
    kind: "manual",
    person: sourcePerson(order, resolveUser),
    detail: CREATED_MANUALLY,
    addedBy: { kind: "manual" },
  };
}

function addedByForIntakeOrder(order: FactoriesWorkOrder, intakeName: string): SplitRunAddedBy {
  if (order.createdBy?.automation) {
    return { kind: "intake", name: intakeName };
  }
  const personName = order.createdBy?.user?.name?.trim();
  if (personName) {
    return { kind: "imported", personName };
  }
  return { kind: "intake", name: intakeName };
}

export function isOriginTicketArtifact(artifact: FactoriesWorkOrderArtifact, source?: SplitRunSource): boolean {
  if (artifact.id?.endsWith("-issue-link")) {
    return true;
  }
  if (source?.kind !== "intake" || !source.ticket) {
    return false;
  }
  return extractArtifactUrl(toArtifactDataRecord(artifact.data)) === source.ticket.href;
}

type SplitRunIntakeSource = Extract<SplitRunSource, { kind: "intake" }>;

function intakeSourceFromHref(
  href: string,
  intakeKind = intakeKindFromHref(href),
  label = sourceTicketLabel(href),
): SplitRunIntakeSource {
  return {
    kind: "intake",
    ...INTAKE_PRESENTATION[intakeKind],
    ticket: { label, href },
  };
}

function intakeKindForAutomation(automation: FactoriesAutomationRef): SplitRunIntakeKind {
  const key = canvasKeyForAutomation({ id: automation.appId, name: automation.appName });
  if (key === "sentry") {
    return "sentry-exceptions";
  }
  if (key === "slack") {
    return "slack";
  }
  return intakeKindFromLabel(`${automation.appId ?? ""} ${automation.appName ?? ""}`);
}

function intakeKindFromLabel(label: string): SplitRunIntakeKind {
  return INTAKE_KIND_HINTS.find((hint) => hint.pattern.test(label))?.kind ?? "github-issues";
}

function intakeSourceFromKind(intakeKind: SplitRunIntakeKind): SplitRunIntakeSource {
  return {
    kind: "intake",
    ...INTAKE_PRESENTATION[intakeKind],
  };
}

function intakeKindFromHref(href: string): SplitRunIntakeKind {
  const parsed = parseUrl(href);
  const host = parsed?.hostname ?? "";
  if (host === "bitbucket.org") {
    return "bitbucket-prs";
  }
  if (host.includes("sentry.io")) {
    return "sentry-exceptions";
  }
  if (host.includes("slack.com")) {
    return "slack";
  }
  if (host.includes("atlassian.net") || host.includes("jira.com")) {
    return "jira-issues";
  }
  if (host === "linear.app" || host.endsWith(".linear.app")) {
    return "linear-issues";
  }
  if (isGitHubDependabotAlertHref(parsed)) {
    return "dependabot-alerts";
  }
  return intakeKindFromLabel(host);
}

function isGitHubDependabotAlertHref(parsed: URL | undefined): boolean {
  return parsed?.hostname === "github.com" && /^\/[^/]+\/[^/]+\/security\/dependabot(?:\/|$)/i.test(parsed.pathname);
}

function sourcePerson(order: FactoriesWorkOrder, resolveUser?: OrgUserDisplayLookup): OrgUserDisplay {
  const user = order.createdBy?.user;
  if (!user?.id && !user?.name) {
    return SOURCE_PERSON_FALLBACK;
  }
  // `resolveUser` looks the source up against the org members list, which
  // carries the avatar image. Without it we can only show initials.
  const resolved = resolveUser?.(user.id, user.name);
  if (resolved) {
    return resolved;
  }
  const name = user.name?.trim() || SOURCE_PERSON_FALLBACK.name;
  return {
    id: user.id ?? SOURCE_PERSON_FALLBACK.id,
    name,
    initials: getUserInitials(name),
    avatarUrl: user.id === SOURCE_PERSON_FALLBACK.id ? SOURCE_PERSON_FALLBACK.avatarUrl : undefined,
  };
}

function parseUrl(url: string): URL | undefined {
  try {
    return new URL(url);
  } catch {
    return undefined;
  }
}

function githubTicketLabel(parsed: URL): string | undefined {
  if (parsed.hostname !== "github.com") {
    return undefined;
  }
  const [, owner, repo, kind, number] = parsed.pathname.split("/");
  if (!owner || !repo || !number || (kind !== "issues" && kind !== "pull")) {
    return undefined;
  }
  return `${owner}/${repo}#${number}`;
}

function bitbucketPullRequestLabel(parsed: URL): string | undefined {
  if (parsed.hostname !== "bitbucket.org") {
    return undefined;
  }
  const [, owner, repo, kind, number] = parsed.pathname.split("/");
  if (!owner || !repo || kind !== "pull-requests" || !number) {
    return undefined;
  }
  return `${owner}/${repo}#${number}`;
}

function linearTicketLabel(parsed: URL): string | undefined {
  if (parsed.hostname !== "linear.app" && !parsed.hostname.endsWith(".linear.app")) {
    return undefined;
  }
  const parts = parsed.pathname.split("/").filter(Boolean);
  const issueAt = parts.indexOf("issue");
  if (issueAt >= 0 && parts[issueAt + 1]) {
    return parts[issueAt + 1];
  }
  return undefined;
}

function jiraTicketLabel(parsed: URL): string | undefined {
  if (!parsed.hostname.includes("atlassian.net") && !parsed.hostname.includes("jira.com")) {
    return undefined;
  }
  const parts = parsed.pathname.split("/").filter(Boolean);
  if (parts.length === 0) {
    return undefined;
  }
  if (parts[0] === "browse" && parts[1]) {
    return parts[1];
  }
  const issuesAt = parts.indexOf("issues");
  if (issuesAt >= 0 && parts[issuesAt + 1]) {
    return parts[issuesAt + 1];
  }
  return parts.at(-1);
}

function hostTicketId(parsed: URL): string | undefined {
  const parts = parsed.pathname.split("/").filter(Boolean);
  if (parsed.hostname.endsWith("sentry.io")) {
    const issuesAt = parts.indexOf("issues");
    return issuesAt >= 0 ? parts[issuesAt + 1] : parts.at(-1);
  }
  if (parsed.hostname.endsWith("pagerduty.com")) {
    const incidentsAt = parts.indexOf("incidents");
    return incidentsAt >= 0 ? parts[incidentsAt + 1] : parts.at(-1);
  }
  if (parsed.hostname.endsWith("slack.com")) {
    const archivesAt = parts.indexOf("archives");
    return archivesAt >= 0 ? parts[archivesAt + 1] : parts.at(-1);
  }
  return parts.at(-1);
}
