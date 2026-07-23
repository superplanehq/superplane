import { Bug, FilePenLine, TestTube2, type LucideIcon } from "lucide-react";

export type FactoryStartingTaskId = "unit-test" | "fix-bug" | "improve-agents-md";

export interface FactoryStartingTask {
  id: FactoryStartingTaskId;
  label: string;
  icon: LucideIcon;
  iconClassName: string;
  prompt: string;
}

export const FACTORY_STARTING_TASKS: FactoryStartingTask[] = [
  {
    id: "unit-test",
    label: "Write test",
    icon: TestTube2,
    iconClassName: "text-amber-500",
    prompt:
      "Scan the codebase to understand its main business logic. Then identify ONE untested function related to this business logic and write a single focused, useful unit test for it. Cover the main execution path and follow existing test patterns. Ensure the test passes.",
  },
  {
    id: "fix-bug",
    label: "Fix bug",
    icon: Bug,
    iconClassName: "text-red-500",
    prompt:
      "Scan through the codebase to identify bugs that look important or impactful. Focus on issues that could affect functionality, performance, or user experience. Once you find a significant bug, create a new branch, implement a fix, write or update tests as needed, and commit the changes with a clear description.",
  },
  {
    id: "improve-agents-md",
    label: "Improve AGENTS.md",
    icon: FilePenLine,
    iconClassName: "text-violet-500",
    prompt:
      "Review the existing AGENTS.md and improve it to help coding agents work more effectively in this repository. Make the guidance specific to this codebase, preserving useful instructions and removing outdated or generic ones. If AGENTS.md doesn't exist, create it.",
  },
];
