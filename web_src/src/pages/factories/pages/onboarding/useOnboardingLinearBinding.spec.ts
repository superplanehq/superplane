import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { writeOnboardingLinearProjects } from "./onboardingLinearProjects";
import { useOnboardingLinearBinding } from "./useOnboardingLinearBinding";

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: () => ({
    data: [{ id: "project-1", name: "Platform" }],
    isPending: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));

describe("useOnboardingLinearBinding", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("restores the stored projects when setup mounts with a ready Linear connection", () => {
    writeOnboardingLinearProjects("factory-1", "linear-1", ["project-1"]);

    const { result } = renderHook(() =>
      useOnboardingLinearBinding("org-1", "factory-1", { id: "linear-1", ready: true }),
    );

    expect(result.current.linearProjectIds).toEqual(["project-1"]);
  });

  it("persists a project chosen during setup", () => {
    const { result } = renderHook(() =>
      useOnboardingLinearBinding("org-1", "factory-1", { id: "linear-1", ready: true }),
    );

    act(() => {
      result.current.toggleLinearProject("project-1");
    });

    expect(result.current.linearProjectIds).toEqual(["project-1"]);
    expect(localStorage.getItem("superplane:onboarding-linear-projects:factory-1:linear-1")).toBe(
      JSON.stringify(["project-1"]),
    );
  });
});
