import type {
  FactoriesFactoryAppTemplateAgent,
  FactoriesFactoryOnboarding,
  OrganizationsIntegration,
} from "@/api-client";
import { renderHook } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "bun:test";
import { useCustomAutomationEditorAgent } from "./useCustomAutomationEditorAgent";

let connections: OrganizationsIntegration[] | undefined;
let isError: boolean;
const useConnectedIntegrations = vi.fn((_organizationId: string, _options?: { enabled?: boolean }) => ({
  data: connections,
  isError,
}));
vi.mock("./useIntegrations", () => ({ useConnectedIntegrations }));

beforeEach(() => {
  connections = undefined;
  isError = false;
  useConnectedIntegrations.mockClear();
});

const provider: OrganizationsIntegration = {
  metadata: { id: "provider", name: "team-models", integrationName: "customLlm" },
  status: { state: "ready" },
};
const integrationAgent: FactoriesFactoryAppTemplateAgent = {
  component: "runnerOpenRouter",
  credentialSource: "integration",
  credentialIntegrationName: "team-models",
};
const hostedAgent = { component: "runnerSuperPlane" };
const subscriptionAgent = { component: "runnerCodex", credentialSource: "subscription" };

type Scenario = {
  name: string;
  onboarding?: FactoriesFactoryOnboarding;
  automationId?: string;
  saved?: FactoriesFactoryAppTemplateAgent;
  needsConnections: boolean;
  initialAgent?: FactoriesFactoryAppTemplateAgent;
  availableAgent?: FactoriesFactoryAppTemplateAgent;
  customProvider?: boolean;
};

const scenarios: Scenario[] = [
  {
    name: "saved OpenRouter with conflicting subscription onboarding",
    automationId: "automation",
    saved: { ...integrationAgent, model: "saved-model" },
    onboarding: { agentHarness: "AGENT_HARNESS_SUPERPLANE", codexSubscription: true },
    needsConnections: true,
    initialAgent: { ...integrationAgent, model: "saved-model" },
    availableAgent: { ...integrationAgent, model: "saved-model" },
    customProvider: true,
  },
  {
    name: "saved Claude with integration onboarding",
    automationId: "automation",
    saved: { component: "runnerClaudeCode", credentialSource: "integration", credentialIntegrationName: "claude" },
    onboarding: { agentIntegrationId: "provider" },
    needsConnections: false,
    initialAgent: {
      component: "runnerClaudeCode",
      credentialSource: "integration",
      credentialIntegrationName: "claude",
    },
    availableAgent: {
      component: "runnerClaudeCode",
      credentialSource: "integration",
      credentialIntegrationName: "claude",
    },
  },
  {
    name: "missing saved agent with integration onboarding",
    automationId: "automation",
    onboarding: { agentIntegrationId: "provider" },
    needsConnections: false,
  },
  {
    name: "new hosted agent with empty IDs",
    automationId: "",
    onboarding: { agentHarness: "AGENT_HARNESS_SUPERPLANE", agentIntegrationId: "" },
    needsConnections: false,
    initialAgent: hostedAgent,
    availableAgent: hostedAgent,
  },
  {
    name: "new subscription agent with an integration ID",
    onboarding: { agentHarness: "AGENT_HARNESS_SUPERPLANE", codexSubscription: true, agentIntegrationId: "provider" },
    needsConnections: false,
    initialAgent: subscriptionAgent,
    availableAgent: subscriptionAgent,
  },
  {
    name: "new integration-backed OpenRouter agent",
    onboarding: { agentHarness: "AGENT_HARNESS_CLAUDE_CODE", agentIntegrationId: "provider" },
    needsConnections: true,
    availableAgent: integrationAgent,
    customProvider: true,
  },
  {
    name: "new hosted harness with an explicit integration ID",
    onboarding: { agentHarness: "AGENT_HARNESS_SUPERPLANE", agentIntegrationId: "provider" },
    needsConnections: true,
    availableAgent: integrationAgent,
    customProvider: true,
  },
  {
    name: "new agent with an unspecified harness and empty integration ID",
    onboarding: { agentIntegrationId: "" },
    needsConnections: true,
    availableAgent: integrationAgent,
    customProvider: true,
  },
  { name: "new agent without onboarding", needsConnections: false },
];

it.each(scenarios)("preserves selection and readiness for $name across connection states", (scenario) => {
  const { result, rerender } = renderHook(() =>
    useCustomAutomationEditorAgent("org", scenario.onboarding, scenario.automationId, scenario.saved),
  );
  expect(result.current).toEqual({
    agent: scenario.initialAgent,
    customProvider: false,
    ready: !!scenario.initialAgent && !scenario.needsConnections,
  });
  expect(useConnectedIntegrations).toHaveBeenLastCalledWith("org", { enabled: scenario.needsConnections });

  connections = [provider];
  rerender();
  expect(result.current).toEqual({
    agent: scenario.availableAgent,
    customProvider: scenario.customProvider ?? false,
    ready: !!scenario.availableAgent,
  });

  // A failed refresh can retain cached connections, but cannot make required connections ready.
  isError = true;
  rerender();
  expect(result.current.ready).toBe(!!scenario.availableAgent && !scenario.needsConnections);
  expect(result.current.agent).toEqual(scenario.availableAgent);

  connections = undefined;
  rerender();
  expect(result.current.agent).toEqual(scenario.initialAgent);
  expect(result.current.ready).toBe(!!scenario.initialAgent && !scenario.needsConnections);
});

it("keeps a saved OpenRouter agent unready when its named provider is missing", () => {
  connections = [{ ...provider, metadata: { ...provider.metadata, name: "other-models" } }];
  const { result } = renderHook(() => useCustomAutomationEditorAgent("org", undefined, "automation", integrationAgent));
  expect(result.current).toEqual({ agent: integrationAgent, customProvider: false, ready: false });
});
