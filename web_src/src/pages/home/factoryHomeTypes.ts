import type { ComponentProps } from "react";

import type { RunStatusBadge } from "@/ui/Runs/RunStatusBadge";

export type FactoryRunStatus = ComponentProps<typeof RunStatusBadge>["status"];

/** Overall health of the factory for a project, driving the header state. */
export type FactoryHealth = "healthy" | "degraded" | "paused";

export interface FactoryIntegration {
  id: string;
  label: string;
  connected: boolean;
}

/** Identity and ownership block — who owns this factory and what it acts on. */
export interface FactoryProject {
  name: string;
  description: string;
  repository: string;
  repositoryUrl: string;
  defaultBranch: string;
  owner: string;
  health: FactoryHealth;
  /** Shown when health is not `healthy`, with the action that resolves it. */
  healthDetail?: string;
  integrations: FactoryIntegration[];
}

/** A prompt the factory can be kicked off with, mirroring `factory.json`. */
export interface FactoryStartingTask {
  id: string;
  label: string;
  description: string;
}

export interface FactoryPullRequest {
  number: number;
  url: string;
}

/** An agent run currently moving through the factory. */
export interface FactoryRun {
  id: string;
  title: string;
  status: FactoryRunStatus;
  /** Human-readable pipeline position, e.g. "Opening pull request". */
  stage: string;
  branch: string;
  startedAt: string;
  pullRequest?: FactoryPullRequest;
}

/** A run parked waiting on a person — the factory's approval queue. */
export interface FactoryReviewItem {
  id: string;
  title: string;
  /** Why it stopped, e.g. "Agent requested approval to merge". */
  reason: string;
  waitingSince: string;
  pullRequest?: FactoryPullRequest;
}

export interface FactoryOutcomeMetric {
  id: string;
  label: string;
  value: string;
  /** Signed change against `deltaPeriod`, e.g. "+12%". Omit when not comparable. */
  delta?: string;
  deltaPeriod?: string;
  /** Which direction of `delta` counts as an improvement. */
  betterDirection: "up" | "down";
  /** Omit before the factory has enough history — a flat line would imply zeroes. */
  trend?: number[];
}

export interface FactoryActivityEntry {
  id: string;
  summary: string;
  actor: string;
  at: string;
}

export interface FactoryHomeData {
  project: FactoryProject;
  startingTasks: FactoryStartingTask[];
  needsReview: FactoryReviewItem[];
  inFlight: FactoryRun[];
  outcomes: FactoryOutcomeMetric[];
  activity: FactoryActivityEntry[];
}
