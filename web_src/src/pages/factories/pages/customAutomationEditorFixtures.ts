import type { FactoriesFactoryCustomAutomation } from "@/api-client";

export const reviewSource = `apiVersion: v2
kind: Automation
name: Review
triggers: [{use: manual}]
steps: [{name: Review, prompt: Review the changes.}]
`;

export const reviewAutomation: FactoriesFactoryCustomAutomation = {
  automation: { id: "automation", name: "Repository review" },
  settings: {
    startMode: "START_MODE_MANUAL",
    agent: { component: "runnerClaudeCode" },
    steps: [
      { name: "Lint", type: "TYPE_BASH", command: "make lint", workingDirectory: "repo", secrets: ["deploy"] },
      { name: "Review", type: "TYPE_PROMPT", prompt: "Review the repository.", workingDirectory: "repo" },
    ],
  },
};

export const savedHostedAutomation: FactoriesFactoryCustomAutomation = {
  settings: { agent: { component: "runnerSuperPlane", model: "hosted::openai::anthropic/claude-test" } },
};
