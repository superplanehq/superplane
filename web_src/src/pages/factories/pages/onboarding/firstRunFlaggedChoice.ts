import type { IssuesChoiceId } from "./onboardingFixtures";

type FlaggedIssuesChoice = "vcs" | "jira" | "linear";

function shouldClearSavedFlaggedChoice(args: {
  issuesChoice: IssuesChoiceId | null;
  source: FlaggedIssuesChoice;
  featureLoading: boolean;
  available: boolean;
  organizationReady: boolean;
}): boolean {
  if (args.featureLoading || args.available || args.issuesChoice !== args.source) return false;
  return args.organizationReady;
}

export function shouldClearSavedJiraChoice(args: {
  issuesChoice: IssuesChoiceId | null;
  featureLoading: boolean;
  jiraAvailable: boolean;
  organizationReady: boolean;
}): boolean {
  return shouldClearSavedFlaggedChoice({
    issuesChoice: args.issuesChoice,
    source: "jira",
    featureLoading: args.featureLoading,
    available: args.jiraAvailable,
    organizationReady: args.organizationReady,
  });
}

export function shouldClearSavedVcsChoice(args: {
  issuesChoice: IssuesChoiceId | null;
  featureLoading: boolean;
  vcsAvailable: boolean;
  organizationReady: boolean;
}): boolean {
  return shouldClearSavedFlaggedChoice({
    issuesChoice: args.issuesChoice,
    source: "vcs",
    featureLoading: args.featureLoading,
    available: args.vcsAvailable,
    organizationReady: args.organizationReady,
  });
}

export function shouldClearSavedLinearChoice(args: {
  issuesChoice: IssuesChoiceId | null;
  featureLoading: boolean;
  linearAvailable: boolean;
  organizationReady: boolean;
}): boolean {
  return shouldClearSavedFlaggedChoice({
    issuesChoice: args.issuesChoice,
    source: "linear",
    featureLoading: args.featureLoading,
    available: args.linearAvailable,
    organizationReady: args.organizationReady,
  });
}

export type SavedFlaggedChoiceBlock = "loading" | "lookup-failed";

function savedFlaggedChoiceBlock(args: {
  issuesChoice: IssuesChoiceId | null;
  source: FlaggedIssuesChoice;
  featureLoading: boolean;
  available: boolean;
  organizationReady: boolean;
}): SavedFlaggedChoiceBlock | null {
  if (args.available || args.issuesChoice !== args.source) return null;
  if (args.featureLoading) return "loading";
  if (!args.organizationReady) return "lookup-failed";
  return null;
}

/** A saved Jira choice cannot continue until the feature lookup confirms Jira. */
export function savedJiraChoiceBlock(args: {
  issuesChoice: IssuesChoiceId | null;
  featureLoading: boolean;
  jiraAvailable: boolean;
  organizationReady: boolean;
}): SavedFlaggedChoiceBlock | null {
  return savedFlaggedChoiceBlock({
    issuesChoice: args.issuesChoice,
    source: "jira",
    featureLoading: args.featureLoading,
    available: args.jiraAvailable,
    organizationReady: args.organizationReady,
  });
}

/** A saved VCS choice cannot continue until the intake catalog confirms it. */
export function savedVcsChoiceBlock(args: {
  issuesChoice: IssuesChoiceId | null;
  featureLoading: boolean;
  vcsAvailable: boolean;
  organizationReady: boolean;
}): SavedFlaggedChoiceBlock | null {
  return savedFlaggedChoiceBlock({
    issuesChoice: args.issuesChoice,
    source: "vcs",
    featureLoading: args.featureLoading,
    available: args.vcsAvailable,
    organizationReady: args.organizationReady,
  });
}

/** A saved Linear choice cannot continue until the feature lookup confirms Linear. */
export function savedLinearChoiceBlock(args: {
  issuesChoice: IssuesChoiceId | null;
  featureLoading: boolean;
  linearAvailable: boolean;
  organizationReady: boolean;
}): SavedFlaggedChoiceBlock | null {
  return savedFlaggedChoiceBlock({
    issuesChoice: args.issuesChoice,
    source: "linear",
    featureLoading: args.featureLoading,
    available: args.linearAvailable,
    organizationReady: args.organizationReady,
  });
}
