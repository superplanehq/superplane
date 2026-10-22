import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { EMPTY_LINEAR_INTAKE_SCOPE, type LinearIntakeScope } from "../linearIntakeScope";
import { writeOnboardingLinearScope } from "./onboardingLinearScope";
import { useOnboardingLinearBinding } from "./useOnboardingLinearBinding";

type Resources = { data: { id: string; name: string }[]; isPending: boolean; isError: boolean };

const linearResources = vi.hoisted(() => ({
  team: { data: [{ id: "team-1", name: "Platform" }], isPending: false, isError: false } as Resources,
  project: { data: [{ id: "project-1", name: "Checkout" }], isPending: false, isError: false } as Resources,
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: (_organizationId: string, _integrationId: string, resourceType: string) => ({
    ...(resourceType === "team" ? linearResources.team : linearResources.project),
    refetch: vi.fn(),
  }),
}));

const STORAGE_KEY = "superplane:onboarding-linear-scope:factory-1:linear-1";
const platformScope: LinearIntakeScope = { ...EMPTY_LINEAR_INTAKE_SCOPE, teamId: "team-1" };

function renderBinding() {
  return renderHook(() => useOnboardingLinearBinding("org-1", "factory-1", { id: "linear-1", ready: true }));
}

describe("useOnboardingLinearBinding", () => {
  beforeEach(() => {
    localStorage.clear();
    linearResources.team = { data: [{ id: "team-1", name: "Platform" }], isPending: false, isError: false };
    linearResources.project = { data: [{ id: "project-1", name: "Checkout" }], isPending: false, isError: false };
  });

  it("restores the stored scope when setup mounts with a ready Linear connection", () => {
    const stored = { teamId: "team-1", projectIds: ["project-1"], includeIssuesWithoutProject: false, labels: ["bug"] };
    writeOnboardingLinearScope("factory-1", "linear-1", stored);

    const { result } = renderBinding();

    expect(result.current.linearScope).toEqual(stored);
  });

  it("persists a team chosen during setup", () => {
    const { result } = renderBinding();

    act(() => {
      result.current.changeLinearScope(platformScope);
    });

    expect(result.current.linearScope).toEqual(platformScope);
    expect(JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "null")).toEqual(platformScope);
  });

  it("drops a saved team that Linear no longer returns", () => {
    writeOnboardingLinearScope("factory-1", "linear-1", { ...platformScope, teamId: "team-gone" });

    const { result } = renderBinding();

    expect(result.current.linearScope).toEqual(EMPTY_LINEAR_INTAKE_SCOPE);
    expect(localStorage.getItem(STORAGE_KEY)).toBeNull();
  });

  it("drops a saved project that Linear no longer returns for the team", () => {
    writeOnboardingLinearScope("factory-1", "linear-1", {
      ...platformScope,
      projectIds: ["project-1", "project-gone"],
    });

    const { result } = renderBinding();

    expect(result.current.linearScope).toEqual({ ...platformScope, projectIds: ["project-1"] });
    expect(JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "null")).toEqual({
      ...platformScope,
      projectIds: ["project-1"],
    });
  });

  it("keeps the saved scope until Linear returns the current lists", () => {
    const stored = { ...platformScope, projectIds: ["project-gone"] };
    writeOnboardingLinearScope("factory-1", "linear-1", stored);
    linearResources.team = { data: [], isPending: true, isError: false };
    linearResources.project = { data: [], isPending: true, isError: false };

    const { result } = renderBinding();

    expect(result.current.linearScope).toEqual(stored);
  });

  it("keeps the saved scope when Linear cannot return the lists", () => {
    const stored = { ...platformScope, projectIds: ["project-gone"] };
    writeOnboardingLinearScope("factory-1", "linear-1", stored);
    linearResources.team = { data: [], isPending: false, isError: true };
    linearResources.project = { data: [], isPending: false, isError: true };

    const { result } = renderBinding();

    expect(result.current.linearScope).toEqual(stored);
    expect(JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "null")).toEqual(stored);
  });
});
