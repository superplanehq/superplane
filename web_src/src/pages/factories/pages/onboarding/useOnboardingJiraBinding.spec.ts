import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { writeOnboardingJiraProject } from "./onboardingJiraProject";
import { useOnboardingJiraBinding } from "./useOnboardingJiraBinding";

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: () => ({
    data: [{ id: "PAY", name: "Payments" }],
    isPending: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));

describe("useOnboardingJiraBinding", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("restores the stored project when setup mounts with a ready Jira connection", () => {
    writeOnboardingJiraProject("factory-1", "jira-1", "PAY");

    const { result } = renderHook(() =>
      useOnboardingJiraBinding("org-1", "factory-1", { id: "jira-1", ready: true }),
    );

    expect(result.current.jiraProjectId).toBe("PAY");
  });

  it("persists a project chosen during setup", () => {
    const { result } = renderHook(() =>
      useOnboardingJiraBinding("org-1", "factory-1", { id: "jira-1", ready: true }),
    );

    act(() => {
      result.current.setJiraProjectId("CORE");
    });

    expect(result.current.jiraProjectId).toBe("CORE");
    expect(localStorage.getItem("superplane:onboarding-jira-project:factory-1:jira-1")).toBe("CORE");
  });
});
