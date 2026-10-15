import type { FactoriesFactoryAgentResource } from "@/api-client";

export const CUSTOM_SYSTEM_SKILL: FactoriesFactoryAgentResource = {
  id: "resource-system-refine",
  kind: "KIND_SKILL",
  name: "refine-task",
  enabled: true,
  markdown:
    "---\nname: refine-task\ntitle: Refine task\ndescription: Refine a task into a plan.\n---\n\nMy custom instructions.",
  systemSkill: { templateId: "refine-task", version: 1, latestVersion: 2, state: "STATE_CUSTOM" },
};

export const LATEST_SYSTEM_SKILL: FactoriesFactoryAgentResource = {
  ...CUSTOM_SYSTEM_SKILL,
  markdown:
    "---\nname: refine-task\ntitle: Refine task\ndescription: Refine a task into a plan.\n---\n\nStandard refinement instructions.",
  systemSkill: { templateId: "refine-task", version: 2, latestVersion: 2, state: "STATE_LATEST" },
};
