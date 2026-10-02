import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { writeOnboardingLinearProjects } from "./onboardingLinearProjects";
import { useOnboardingLinearBinding } from "./useOnboardingLinearBinding";

const linearResources = vi.hoisted(() => ({
  data: [{ id: "project-1", name: "Platform" }] as { id: string; name: string }[],
  isPending: false,
  isError: false,
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: () => ({
    data: linearResources.data,
    isPending: linearResources.isPending,
    isError: linearResources.isError,
    refetch: vi.fn(),
  }),
}));

describe("useOnboardingLinearBinding", () => {
  beforeEach(() => {
    localStorage.clear();
    linearResources.data = [{ id: "project-1", name: "Platform" }];
    linearResources.isPending = false;
    linearResources.isError = false;
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

  it("drops a saved project that Linear no longer returns", () => {
    writeOnboardingLinearProjects("factory-1", "linear-1", ["project-1", "project-gone"]);

    const { result } = renderHook(() =>
      useOnboardingLinearBinding("org-1", "factory-1", { id: "linear-1", ready: true }),
    );

    expect(result.current.linearProjectIds).toEqual(["project-1"]);
    expect(localStorage.getItem("superplane:onboarding-linear-projects:factory-1:linear-1")).toBe(
      JSON.stringify(["project-1"]),
    );
  });

  it("keeps a saved project until Linear returns the current list", () => {
    writeOnboardingLinearProjects("factory-1", "linear-1", ["project-gone"]);
    linearResources.isPending = true;
    linearResources.data = [];

    const { result } = renderHook(() =>
      useOnboardingLinearBinding("org-1", "factory-1", { id: "linear-1", ready: true }),
    );

    expect(result.current.linearProjectIds).toEqual(["project-gone"]);
  });
});
