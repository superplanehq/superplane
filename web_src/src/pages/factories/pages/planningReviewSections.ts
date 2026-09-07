import { Cpu, Gauge, KeyRound, ListOrdered, Variable, type LucideIcon } from "lucide-react";

export type PlanningReviewSectionId = "steps" | "runner" | "credentials" | "environment" | "concurrency";

export interface PlanningReviewSection {
  id: PlanningReviewSectionId;
  label: string;
  /** One line under the panel title. It says what the group controls. */
  description: string;
  icon: LucideIcon;
  /** Names from PLANNING_REVIEW_RUNNER_FIELDS, in the order they appear. */
  fieldNames: string[];
}

export const PLANNING_REVIEW_SECTIONS: PlanningReviewSection[] = [
  {
    id: "steps",
    label: "Steps",
    description: "The agent runs these steps in order on the runner.",
    icon: ListOrdered,
    fieldNames: [],
  },
  {
    id: "runner",
    label: "Runner",
    description: "Where the agent runs, which model it uses, and when it times out.",
    icon: Cpu,
    fieldNames: ["machineType", "model", "workingDirectory", "executionTimeoutSeconds"],
  },
  {
    id: "credentials",
    label: "Credentials",
    description: "The account the agent uses to call the model.",
    icon: KeyRound,
    fieldNames: ["credentials"],
  },
  {
    id: "environment",
    label: "Environment variables",
    description: "Values passed into every step of the agent.",
    icon: Variable,
    fieldNames: ["environmentFrom", "environment"],
  },
  {
    id: "concurrency",
    label: "Concurrency",
    description: "How many executions of this agent can run at the same time.",
    icon: Gauge,
    fieldNames: [],
  },
];

export const PLANNING_REVIEW_DEFAULT_SECTION: PlanningReviewSectionId = "steps";
