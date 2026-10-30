import { storybookHostedLlmModels } from "@/pages/home/__fixtures__/hostedLlmModels";
import type { FactoriesFixture } from "./factoryPageResponses";
import { ORGANIZATION_USERS, STORYBOOK_ME_USER_ID } from "./factoryPageIds";
import type { FactoriesRoute } from "./handlers";

const pattern = (suffix: string) => new RegExp(`^/api/v1/factories/([^/]+)/${suffix}$`);

function stringIds(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value.filter((id): id is string => typeof id === "string");
}

function catalogIds(fixture: FactoriesFixture, provider: string, source: string) {
  if (source === "subscription") return fixture.codexConnection?.availableModels ?? [];
  if (source === "byok" && fixture.byokCandidatesByProvider?.[provider])
    return fixture.byokCandidatesByProvider[provider];
  return storybookHostedLlmModels(provider).models.map((model) => model.id);
}

function policyResponse(fixture: FactoriesFixture, key: string, catalog: string[]) {
  const policy = fixture.workspaceModelPolicies?.[key];
  const inheritParent = policy?.inheritParent ?? true;
  const configuredModels = policy?.configuredModels ?? [];
  const selected = inheritParent ? catalog : catalog.filter((id) => configuredModels.includes(id));
  const models = (ids: string[]) => ids.map((id) => ({ id, name: id }));
  return {
    json: {
      parent: models(catalog),
      selected: models(selected),
      configuredModels: models(configuredModels),
      inheritParent,
    },
  };
}

function workspaceModelPolicyRoute(fixture: FactoriesFixture): FactoriesRoute {
  return {
    pattern: pattern("llm-models"),
    resolve: (match, method, body, url) => {
      if (fixture.workspaceModelPolicyError)
        return { status: 503, json: { message: fixture.workspaceModelPolicyError } };
      const provider = url.searchParams.get("provider") || String(body?.provider ?? "");
      const source = url.searchParams.get("fundingSource") || String(body?.fundingSource ?? "hosted");
      const key = `${match[1]}:${source}:${provider}`;
      fixture.workspaceModelPolicies ??= {};
      if (method === "PUT")
        fixture.workspaceModelPolicies[key] = {
          inheritParent:
            typeof body?.inheritParent === "boolean" ? body.inheritParent : stringIds(body?.allowedModels).length === 0,
          configuredModels: stringIds(body?.allowedModels),
        };
      return policyResponse(fixture, key, catalogIds(fixture, provider, source));
    },
  };
}

function personalConnectionRoutes(fixture: FactoriesFixture): FactoriesRoute[] {
  return [
    {
      pattern: pattern("owner-eligibility"),
      resolve: () => ({
        json: {
          subscription: true,
          ownershipChangeAllowed: true,
          members: ORGANIZATION_USERS.map((user) => ({
            userId: user.id,
            eligible: user.id === STORYBOOK_ME_USER_ID && fixture.codexConnection?.status === "ready",
          })),
        },
      }),
    },
    {
      pattern: /\/api\/v1\/me\/codex-connection$/,
      resolve: (_match, method, body) => {
        fixture.codexConnection ??= { eligible: true, status: "disconnected" };
        if (method === "DELETE") fixture.codexConnection = { eligible: true, status: "disconnected" };
        if (method === "POST" && body?.refreshCatalog !== true)
          fixture.codexConnection = {
            eligible: true,
            status: "pending",
            verificationUrl: "https://auth.openai.com/codex/device",
            verificationCode: "ABCD-1234",
          };
        return { json: { connection: fixture.codexConnection } };
      },
    },
    {
      pattern: /\/api\/v1\/me\/codex-connection:cancel$/,
      resolve: () => {
        fixture.codexConnection = { eligible: true, status: "disconnected" };
        return { json: { connection: fixture.codexConnection } };
      },
    },
  ];
}

function harnessForSource(source: string) {
  if (source === "hosted") return "AGENT_HARNESS_SUPERPLANE" as const;
  if (source === "openai" || source === "subscription") return "AGENT_HARNESS_CODEX" as const;
  return "AGENT_HARNESS_CLAUDE_CODE" as const;
}

function modelSourceRoute(fixture: FactoriesFixture): FactoriesRoute {
  return {
    pattern: pattern("model-source"),
    resolve: (match, _method, body) => {
      const factory = fixture.factories.find((entry) => entry.id === match[1]);
      if (!factory) return { status: 404, json: {} };
      const source = String(body?.source ?? "hosted");
      const ownKey = source !== "hosted" && source !== "subscription";
      factory.onboarding = {
        ...factory.onboarding,
        codexSubscription: source === "subscription",
        codexDefaultOwnerId:
          source === "subscription" ? String(body?.defaultOwnerId ?? "") : factory.onboarding?.codexDefaultOwnerId,
        agentHarness: harnessForSource(source),
        agentIntegrationId: ownKey ? `int-${source}` : undefined,
      };
      if (ownKey) fixture.byokConnectedProviders = [...new Set([...(fixture.byokConnectedProviders ?? []), source])];
      return { json: { factory } };
    },
  };
}

export function factoryModelSettingsRoutes(fixture: FactoriesFixture): FactoriesRoute[] {
  return [workspaceModelPolicyRoute(fixture), ...personalConnectionRoutes(fixture), modelSourceRoute(fixture)];
}
