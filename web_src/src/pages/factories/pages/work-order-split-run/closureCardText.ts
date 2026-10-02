import type { FactoriesFactoryPullRequest } from "@/api-client";
import { safeExternalUrl } from "@/lib/safeExternalUrl";

import { pullRequestListLabel, pullRequestState } from "../../lib/workOrderPullRequest";
import type { WorkOrderDisplayStatus } from "../../lib/workOrderProgress";
import { doneFooterForStatus } from "./splitRunFooter";
import type { SplitRunFooterCloser } from "./splitRunFooterActor";

/**
 * Why this task closed, for the Done Completed card. A merged pull
 * request is the reason when one exists. The person who merged is a
 * link when SuperPlane stored a profile URL.
 */
export function closureCardDescription(
  status: WorkOrderDisplayStatus,
  closer?: SplitRunFooterCloser,
  pullRequests: FactoriesFactoryPullRequest[] = [],
): string | undefined {
  const pullRequest = latestMergedPullRequest(pullRequests);
  const person = closerPerson(closer);
  if (status === "completed" && pullRequest) {
    const request = pullRequestMarkdown(pullRequest);
    if (person) {
      return `Resolved because ${personMarkdown(person)} merged ${request}.`;
    }
    return `Resolved because ${request} is merged.`;
  }
  return closedFooterSentence(status, closer);
}

function latestMergedPullRequest(pullRequests: FactoriesFactoryPullRequest[]): FactoriesFactoryPullRequest | undefined {
  const merged = pullRequests.filter((pullRequest) => pullRequestState(pullRequest.state) === "merged");
  if (merged.length === 0) {
    return undefined;
  }
  return [...merged].sort((left, right) => Date.parse(right.mergedAt ?? "") - Date.parse(left.mergedAt ?? ""))[0];
}

function closerPerson(closer?: SplitRunFooterCloser): { name: string; href?: string } | undefined {
  const actorName = closer?.actor?.name?.trim();
  if (actorName) {
    return { name: actorName, ...(closer.actorHref ? { href: closer.actorHref } : {}) };
  }
  const automationName = closer?.automationName?.trim();
  if (!automationName) {
    return undefined;
  }
  if (closer.automationHref) {
    return { name: automationName, href: closer.automationHref };
  }
  if (/\s/.test(automationName)) {
    return undefined;
  }
  return { name: automationName };
}

function pullRequestMarkdown(pullRequest: FactoriesFactoryPullRequest): string {
  return markdownLink(pullRequestListLabel(pullRequest), pullRequest.url);
}

function personMarkdown(person: { name: string; href?: string }): string {
  return markdownLink(person.name, person.href);
}

function markdownLink(label: string, href: string | null | undefined): string {
  const text = escapeMarkdownLinkLabel(label);
  const safe = safeExternalUrl(href);
  if (!safe) {
    return text;
  }
  return `[${text}](${safe})`;
}

function escapeMarkdownLinkLabel(label: string): string {
  return label.replace(/\[/g, "\\[").replace(/\]/g, "\\]");
}

function closedFooterSentence(status: WorkOrderDisplayStatus, closer?: SplitRunFooterCloser): string | undefined {
  const note = doneFooterForStatus(status, closer).note;
  if (!note) {
    return undefined;
  }
  return `${note.actor?.name ? `${note.actor.name} ` : ""}${note.headline}.`;
}
