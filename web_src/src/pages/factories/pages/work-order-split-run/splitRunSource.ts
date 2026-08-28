import type { FactoriesAutomationRef, FactoriesWorkOrder, FactoriesWorkOrderArtifact } from "@/api-client";
import githubIcon from "@/assets/icons/integrations/github.svg";
import pagerdutyIcon from "@/assets/icons/integrations/pagerduty.svg";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import slackIcon from "@/assets/icons/integrations/slack.svg";
import { getUserInitials, type OrgUserDisplay } from "@/lib/orgUserDisplay";

import {
  STORYBOOK_ME_USER_AVATAR_URL,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
} from "../../__fixtures__/factoryPageResponses";
import { extractArtifactUrl, toArtifactDataRecord } from "../../lib/workOrderArtifact";
import { reviewCandidateForWorkOrderId } from "../onboarding/first-run/reviewCandidates";
import { canvasKeyForAutomation } from "./splitRunCanvases";

export const CREATED_MANUALLY = "Created manually";

export type SplitRunIntakeKind = "github-issues" | "sentry-exceptions" | "pagerduty-incidents" | "slack";

export type SplitRunSource =
  | {
      kind: "intake";
      name: string;
      iconSrc: string;
      iconAlt: string;
      ticket?: { label: string; href: string };
    }
  | {
      kind: "manual";
      person: OrgUserDisplay;
      detail: typeof CREATED_MANUALLY;
    };

const SOURCE_PERSON_FALLBACK: OrgUserDisplay = {
  id: STORYBOOK_ME_USER_ID,
  name: STORYBOOK_ME_USER_NAME,
  initials: getUserInitials(STORYBOOK_ME_USER_NAME),
  avatarUrl: STORYBOOK_ME_USER_AVATAR_URL,
};

const INTAKE_PRESENTATION: Record<SplitRunIntakeKind, { name: string; iconSrc: string; iconAlt: string }> = {
  "github-issues": { name: "GitHub issues", iconSrc: githubIcon, iconAlt: "GitHub" },
  "sentry-exceptions": { name: "Sentry exceptions", iconSrc: sentryIcon, iconAlt: "Sentry" },
  "pagerduty-incidents": { name: "PagerDuty incidents", iconSrc: pagerdutyIcon, iconAlt: "PagerDuty" },
  slack: { name: "Slack", iconSrc: slackIcon, iconAlt: "Slack" },
};

export function sourceTicketLabel(url: string): string {
  const parsed = parseUrl(url);
  if (!parsed) {
    return url;
  }
  const github = githubTicketLabel(parsed);
  if (github) {
    return github;
  }
  const org = parsed.hostname.split(".")[0] ?? parsed.hostname;
  const id = hostTicketId(parsed);
  if (id) {
    return `${org}#${id}`;
  }
  return parsed.hostname;
}

export function splitRunIntakeSource(href: string, intakeKind?: SplitRunIntakeKind): SplitRunSource {
  return intakeSourceFromHref(href, intakeKind);
}

export function splitRunSourceForOrder(order: FactoriesWorkOrder): SplitRunSource {
  const originHref = order.origin?.url?.trim();
  if (originHref) {
    return intakeSourceFromHref(
      originHref,
      intakeKindFromHref(originHref),
      order.origin?.label?.trim() || sourceTicketLabel(originHref),
    );
  }

  const candidate = reviewCandidateForWorkOrderId(order.id);
  if (candidate?.issue.url) {
    return intakeSourceFromHref(candidate.issue.url);
  }

  const automation = order.createdBy?.automation;
  if (automation) {
    return intakeSourceFromKind(intakeKindForAutomation(automation));
  }

  return {
    kind: "manual",
    person: sourcePerson(order),
    detail: CREATED_MANUALLY,
  };
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

function intakeSourceFromHref(
  href: string,
  intakeKind = intakeKindFromHref(href),
  label = sourceTicketLabel(href),
): SplitRunSource {
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
  if (/pagerduty/i.test(`${automation.appId ?? ""} ${automation.appName ?? ""}`)) {
    return "pagerduty-incidents";
  }
  return "github-issues";
}

function intakeSourceFromKind(intakeKind: SplitRunIntakeKind): SplitRunSource {
  return {
    kind: "intake",
    ...INTAKE_PRESENTATION[intakeKind],
  };
}

function intakeKindFromHref(href: string): SplitRunIntakeKind {
  const host = parseUrl(href)?.hostname ?? "";
  if (host.includes("sentry.io")) {
    return "sentry-exceptions";
  }
  if (host.includes("pagerduty.com")) {
    return "pagerduty-incidents";
  }
  if (host.includes("slack.com")) {
    return "slack";
  }
  return "github-issues";
}

function sourcePerson(order: FactoriesWorkOrder): OrgUserDisplay {
  const user = order.createdBy?.user;
  if (!user?.id && !user?.name) {
    return SOURCE_PERSON_FALLBACK;
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
