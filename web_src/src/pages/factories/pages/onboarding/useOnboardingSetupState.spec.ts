import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { IntegrationId } from "./onboardingFixtures";
import { useOnboardingSetupState } from "./useOnboardingSetupState";

describe("useOnboardingSetupState", () => {
  it("uses real connected state and completes discovery without fixture counts", () => {
    const connected = new Set<IntegrationId>(["github", "claude"]);
    const { result } = renderHook(() =>
      useOnboardingSetupState("Payments", {
        connected,
        simulateDiscovery: false,
      }),
    );

    act(() => {
      result.current.selectVcsHost("github");
      result.current.selectRepo("acme/payments");
    });
    act(() => result.current.startIssuesDiscovery());

    expect(result.current.vcsReady).toBe(true);
    expect(result.current.repoReady).toBe(true);
    expect(result.current.issuesDiscovered).toBe(true);
    expect(result.current.issuesChoice).toBe("vcs");
    expect(result.current.issueCount).toBeUndefined();
  });
});
