import type {
  FactoriesFactoryPrFeedbackHandler,
  FactoriesFactoryPrFeedbackHandlerSettings,
  FactoriesFactoryPrFeedbackHandlerSource,
  FactoryPrFeedbackHandlerCheckSettings,
  FactoryPrFeedbackHandlerConflictSettings,
} from "@/api-client";
import githubIcon from "@/assets/icons/integrations/github.svg";

export type PRFeedbackSettingsTab = "general" | "automation";
export type PRFeedbackSourceId = "discussion" | "checks" | "conflicts";

export function isPRFeedbackSettingsTab(value: string | null | undefined): value is PRFeedbackSettingsTab {
  return value === "general" || value === "automation";
}

export interface PRFeedbackSource {
  id: PRFeedbackSourceId;
  name: string;
  description: string;
  listenTitle: string;
  iconSrc: string;
  iconAlt: string;
  defaultName: string;
}

export const PR_FEEDBACK_SOURCES: PRFeedbackSource[] = [
  {
    id: "discussion",
    name: "Pull request discussion",
    description: "Address comments and reviews after a mention.",
    listenTitle: "Listening to pull request comments",
    iconSrc: githubIcon,
    iconAlt: "GitHub",
    defaultName: "Address PR feedback",
  },
  {
    id: "checks",
    name: "Pull request checks",
    description: "Wait for checks and start one agent run when selected checks fail.",
    listenTitle: "Monitoring pull request checks",
    iconSrc: githubIcon,
    iconAlt: "GitHub",
    defaultName: "Fix pull request checks",
  },
  {
    id: "conflicts",
    name: "Pull request conflicts",
    description: "Wait for merge conflicts and start one agent run when GitHub reports a conflict.",
    listenTitle: "Monitoring pull request conflicts",
    iconSrc: githubIcon,
    iconAlt: "GitHub",
    defaultName: "Resolve pull request conflicts",
  },
];

export function prFeedbackSourceById(id: string | undefined): PRFeedbackSource | undefined {
  return PR_FEEDBACK_SOURCES.find((source) => source.id === id);
}

export function prFeedbackSourceId(source?: FactoriesFactoryPrFeedbackHandlerSource): PRFeedbackSourceId {
  if (source === "SOURCE_PULL_REQUEST_CHECKS") {
    return "checks";
  }
  if (source === "SOURCE_PULL_REQUEST_CONFLICTS") {
    return "conflicts";
  }
  return "discussion";
}

export function apiPRFeedbackSource(sourceId: PRFeedbackSourceId): FactoriesFactoryPrFeedbackHandlerSource {
  if (sourceId === "checks") {
    return "SOURCE_PULL_REQUEST_CHECKS";
  }
  if (sourceId === "conflicts") {
    return "SOURCE_PULL_REQUEST_CONFLICTS";
  }
  return "SOURCE_PULL_REQUEST_DISCUSSION";
}

export function takenPRFeedbackSourceIds(
  handlers: Array<{ source?: FactoriesFactoryPrFeedbackHandlerSource }>,
): PRFeedbackSourceId[] {
  return handlers.map((handler) => prFeedbackSourceId(handler.source));
}

export function hasAvailablePRFeedbackSource(taken: readonly PRFeedbackSourceId[]): boolean {
  return PR_FEEDBACK_SOURCES.some((source) => !taken.includes(source.id));
}

export interface PRFeedbackDraftSettings {
  source: PRFeedbackSourceId;
  name: string;
  repository: string;
  mention: string;
  ignoreBots: boolean;
  allowedBots: string[];
  checkNames: string[];
  maximumAttempts: number;
  baseBranch: string;
  runnerIntegrationIds: string[];
}

export const PR_FEEDBACK_SETTINGS_COPY = {
  tabsLabel: "PR feedback settings",
  generalTab: "General",
  automationTab: "Automation",
  nameLabel: "Name",
  nameHelper: "This name appears in the workspace app list.",
  repositoryLabel: "Repository",
  repositoryHelper: "Listen for mentions on pull requests in this repository.",
  checksRepositoryHelper: "Wait for checks on pull requests in this repository.",
  conflictsRepositoryHelper: "Watch pull requests in this repository for merge conflicts.",
  mentionLabel: "Mention",
  mentionHelper: "Use an exact GitHub mention, for example @superplaneagent.",
  ignoreBotsLabel: "Ignore bot comments",
  ignoreBotsHelper: "Do not start a run when a bot writes the mention.",
  allowedBotsLabel: "Allowed bots",
  allowedBotsHelper: "React to comments from these bots even without the mention. Use the bot login.",
  checkNamesLabel: "Status checks",
  checkNamesHelper:
    "SuperPlane waits for each selected check and fixes selected failures. Leave empty to monitor all checks.",
  checkNamesAdd: "Add",
  checkNamesPlaceholder: "lint",
  maximumAttemptsLabel: "Maximum automatic fix attempts",
  maximumAttemptsHelper:
    "SuperPlane pauses automatic fixes after this many consecutive attempts. Passing checks reset the count.",
  conflictsMaximumAttemptsHelper: "SuperPlane pauses automatic conflict fixes after this many consecutive attempts.",
  baseBranchLabel: "Base branch",
  baseBranchHelper: "SuperPlane rechecks open factory pull requests when this branch receives a push.",
  integrationsLabel: "Additional integration access",
  integrationsHelper: "Give the agent access to CI logs from other connected integrations.",
  integrationsMissingBefore: "If this list does not include the integration you need, go to the ",
  integrationsMissingLink: "Integrations page",
  integrationsMissingAfter: " and connect it.",
  integrationsEmpty: "No other connected integrations are available.",
  healthReady: "Ready",
  healthNeedsRepair: "Needs repair",
  healthReadyHelper: "This automation can receive a mention and address it.",
  healthChecksReadyHelper: "This automation can wait for checks and start a fix.",
  healthConflictsReadyHelper: "This automation can wait for merge conflicts and start a fix.",
  healthNeedsRepairHelper: "Open the Automation tab and repair the canvas.",
  save: "Save",
  saving: "Saving",
  delete: "Delete automation",
  deleting: "Deleting",
  keep: "Keep automation",
  confirmDelete: "Delete this automation? This cannot be undone.",
  saveError: "Could not save PR feedback settings.",
  emptyTitle: "No PR feedback automation",
  emptyBody: "Create an automation that addresses pull request feedback.",
  create: "Add feedback handler",
  creating: "Creating",
  createError: "Could not create the PR feedback automation.",
  addHandler: "Add feedback handler",
  pickerTitle: "Add feedback handler",
  pickerDescription: "Choose the signal that starts a pull request feedback run.",
  sourceTaken: "A handler for this source already exists.",
  loading: "Loading PR feedback.",
  loadError: "Could not load PR feedback.",
  retry: "Retry",
  automationEmpty: "This automation has no canvas yet.",
  automationLoading: "Loading the canvas.",
  automationError: "Could not load the canvas.",
  retryAutomation: "Retry",
  editAutomation: "Edit automation",
} as const;

export function prFeedbackDraftFromHandler(handler: FactoriesFactoryPrFeedbackHandler): PRFeedbackDraftSettings {
  const source = prFeedbackSourceId(handler.source);
  const discussion = handler.settings?.discussion;
  const checks = handler.settings?.checks;
  const conflicts = handler.settings?.conflicts;
  return {
    source,
    name: handlerDraftName(handler.name, source),
    repository: handlerDraftRepository(handler),
    mention: handlerDraftMention(discussion?.mention),
    ignoreBots: discussion?.ignoreBots !== false,
    allowedBots: discussion?.allowedBots ?? [],
    checkNames: checks?.names ?? [],
    maximumAttempts: handlerDraftMaximumAttempts(source, checks, conflicts),
    baseBranch: handlerDraftBaseBranch(conflicts?.baseBranch),
    runnerIntegrationIds: checks?.runnerIntegrationIds ?? [],
  };
}

function handlerDraftMaximumAttempts(
  source: PRFeedbackSourceId,
  checks: FactoryPrFeedbackHandlerCheckSettings | undefined,
  conflicts: FactoryPrFeedbackHandlerConflictSettings | undefined,
): number {
  const configured = source === "conflicts" ? conflicts?.maximumAttempts : checks?.maximumAttempts;
  return configured ?? 3;
}

function handlerDraftName(name: string | undefined, source: PRFeedbackSourceId): string {
  const trimmed = name?.trim();
  if (trimmed) {
    return trimmed;
  }
  return prFeedbackSourceById(source)?.defaultName ?? "Address PR feedback";
}

function handlerDraftRepository(handler: FactoriesFactoryPrFeedbackHandler): string {
  return handler.settings?.subject?.repository?.trim() ?? "";
}

function handlerDraftMention(mention: string | undefined): string {
  const trimmed = mention?.trim();
  if (trimmed) {
    return trimmed;
  }
  return "@superplaneagent";
}

function handlerDraftBaseBranch(baseBranch: string | undefined): string {
  const trimmed = baseBranch?.trim();
  if (trimmed) {
    return trimmed;
  }
  return "main";
}

export function prFeedbackListenTitle(source?: FactoriesFactoryPrFeedbackHandlerSource | PRFeedbackSourceId): string {
  const sourceId =
    source === "discussion" || source === "checks" || source === "conflicts" ? source : prFeedbackSourceId(source);
  return prFeedbackSourceById(sourceId)?.listenTitle ?? "Listening to pull request comments";
}

export function normalizePRFeedbackDraft(draft: PRFeedbackDraftSettings): PRFeedbackDraftSettings {
  return {
    source: draft.source,
    name: draft.name.trim(),
    repository: draft.repository.trim(),
    mention: draft.mention.trim(),
    ignoreBots: draft.ignoreBots,
    allowedBots: normalizeAllowedBots(draft.allowedBots),
    checkNames: normalizeUniqueStrings(draft.checkNames),
    maximumAttempts: draft.maximumAttempts,
    baseBranch: draft.baseBranch.trim(),
    runnerIntegrationIds: normalizeUniqueStrings(draft.runnerIntegrationIds),
  };
}

/** Add one exact name. Keep commas inside the name. Skip blanks and duplicates. */
export function appendUniqueTrimmedString(items: string[], raw: string): string[] {
  const name = raw.trim();
  if (name.length === 0 || items.includes(name)) {
    return items;
  }
  return [...items, name];
}

function normalizeAllowedBots(bots: string[]): string[] {
  const seen = new Set<string>();
  const normalized: string[] = [];
  for (const bot of bots) {
    const trimmed = bot.trim().replace(/^@/, "");
    if (!trimmed) {
      continue;
    }
    const key = trimmed.toLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    normalized.push(trimmed);
  }
  return normalized;
}

function normalizeUniqueStrings(values: string[]): string[] {
  const seen = new Set<string>();
  const normalized: string[] = [];
  for (const value of values) {
    const trimmed = value.trim();
    if (!trimmed) {
      continue;
    }
    const key = trimmed.toLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    normalized.push(trimmed);
  }
  return normalized;
}

export function prFeedbackDraftIsValid(draft: PRFeedbackDraftSettings): boolean {
  const next = normalizePRFeedbackDraft(draft);
  if (next.name.length === 0 || next.repository.length === 0) {
    return false;
  }
  if (next.source === "checks") {
    return next.maximumAttempts >= 1 && next.maximumAttempts <= 10;
  }
  if (next.source === "conflicts") {
    return next.baseBranch.length > 0 && next.maximumAttempts >= 1 && next.maximumAttempts <= 10;
  }
  return next.mention.startsWith("@");
}

export function prFeedbackSettingsToApi(draft: PRFeedbackDraftSettings): FactoriesFactoryPrFeedbackHandlerSettings {
  if (draft.source === "checks") {
    return {
      subject: { repository: draft.repository },
      checks: {
        names: draft.checkNames,
        maximumAttempts: draft.maximumAttempts,
        runnerIntegrationIds: draft.runnerIntegrationIds,
      },
    };
  }
  if (draft.source === "conflicts") {
    return {
      subject: { repository: draft.repository },
      conflicts: {
        maximumAttempts: draft.maximumAttempts,
        baseBranch: draft.baseBranch,
      },
    };
  }
  return {
    subject: { repository: draft.repository },
    discussion: {
      mention: draft.mention,
      ignoreBots: draft.ignoreBots,
      allowedBots: draft.allowedBots,
    },
  };
}

// Work-order card labels and activity kinds live in a separate module so
// this file stays focused on draft settings and the API mapping.
export * from "./prFeedbackActivityLabels";
