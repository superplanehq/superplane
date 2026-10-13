import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "bun:test";

import type { IntegrationId } from "./onboardingFixtures";
import { readOnboardingRepoChoice, writeOnboardingRepoChoice } from "./onboardingRepoChoice";
import { writeOnboardingVcsHostChoice } from "./onboardingVcsHostChoice";
import { useOnboardingSetupState } from "./useOnboardingSetupState";

describe("useOnboardingSetupState", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it("keeps a Bitbucket host when setup is created again before a repository is saved", () => {
    const first = renderHook(() =>
      useOnboardingSetupState("Payments", {
        simulateDiscovery: false,
        persistVcsHostKey: "factory-1",
      }),
    );
    act(() => first.result.current.selectVcsHost("bitbucket"));
    first.unmount();

    const second = renderHook(() =>
      useOnboardingSetupState("Payments", {
        connected: new Set<IntegrationId>(["github", "bitbucket"]),
        simulateDiscovery: false,
        persistVcsHostKey: "factory-1",
      }),
    );

    expect(second.result.current.vcsHost).toBe("bitbucket");
  });

  it("restores the selected repository after the page reloads", () => {
    const first = renderHook(() =>
      useOnboardingSetupState("Payments", {
        simulateDiscovery: false,
        persistRepoKey: "factory-1",
      }),
    );
    act(() => {
      first.result.current.selectVcsHost("github");
      first.result.current.selectRepo("acme/payments");
    });
    first.unmount();

    const second = renderHook(() =>
      useOnboardingSetupState("Payments", {
        simulateDiscovery: false,
        persistRepoKey: "factory-1",
      }),
    );

    expect(second.result.current.selectedRepo).toBe("acme/payments");
  });

  it("keeps the saved repository and drops the stored choice", () => {
    writeOnboardingRepoChoice("factory-1", "acme/stored");
    const first = renderHook(() =>
      useOnboardingSetupState("Payments", {
        simulateDiscovery: false,
        persistRepoKey: "factory-1",
        initial: { selectedRepo: "acme/saved" },
      }),
    );

    expect(first.result.current.selectedRepo).toBe("acme/saved");
    expect(readOnboardingRepoChoice("factory-1")).toBeNull();
    first.unmount();

    const second = renderHook(() =>
      useOnboardingSetupState("Payments", {
        simulateDiscovery: false,
        persistRepoKey: "factory-1",
      }),
    );

    expect(second.result.current.selectedRepo).toBeNull();
  });

  it("removes the stored repository when the selection is cleared", () => {
    const first = renderHook(() =>
      useOnboardingSetupState("Payments", {
        simulateDiscovery: false,
        persistRepoKey: "factory-1",
      }),
    );
    act(() => first.result.current.selectRepo("acme/payments"));
    act(() => first.result.current.clearRepository());
    first.unmount();

    const second = renderHook(() =>
      useOnboardingSetupState("Payments", {
        simulateDiscovery: false,
        persistRepoKey: "factory-1",
      }),
    );

    expect(second.result.current.selectedRepo).toBeNull();
  });

  it("removes the stored repository when the host changes", () => {
    const first = renderHook(() =>
      useOnboardingSetupState("Payments", {
        simulateDiscovery: false,
        persistRepoKey: "factory-1",
      }),
    );
    act(() => {
      first.result.current.selectVcsHost("github");
      first.result.current.selectRepo("acme/payments");
      first.result.current.selectVcsHost("bitbucket");
    });
    first.unmount();

    const second = renderHook(() =>
      useOnboardingSetupState("Payments", {
        simulateDiscovery: false,
        persistRepoKey: "factory-1",
      }),
    );

    expect(second.result.current.selectedRepo).toBeNull();
    expect(second.result.current.vcsHost).toBeNull();
  });

  it("keeps the stored repository when the same host is selected again", () => {
    const { result } = renderHook(() =>
      useOnboardingSetupState("Payments", {
        simulateDiscovery: false,
        persistRepoKey: "factory-1",
      }),
    );
    act(() => {
      result.current.selectVcsHost("github");
      result.current.selectRepo("acme/payments");
    });
    act(() => result.current.selectVcsHost("github"));

    expect(result.current.selectedRepo).toBe("acme/payments");
    expect(readOnboardingRepoChoice("factory-1")).toBe("acme/payments");
  });

  it("keeps the workspace host that onboarding already saved", () => {
    writeOnboardingVcsHostChoice("factory-1", "bitbucket");
    const { result } = renderHook(() =>
      useOnboardingSetupState("Payments", {
        simulateDiscovery: false,
        persistVcsHostKey: "factory-1",
        initial: { vcsHost: "github" },
      }),
    );

    expect(result.current.vcsHost).toBe("github");
  });

  it("hydrates saved repository choices only when the state is created", () => {
    const connected = new Set<IntegrationId>(["github"]);
    const { result, rerender } = renderHook(() =>
      useOnboardingSetupState("Payments", {
        connected,
        simulateDiscovery: false,
        initial: {
          vcsHost: "github",
          selectedRepo: "acme/old",
          issuesRepo: "acme/old",
          issuesChoice: "vcs",
        },
      }),
    );

    expect(result.current.selectedRepo).toBe("acme/old");
    act(() => result.current.selectRepo("acme/new"));
    rerender();

    expect(result.current.selectedRepo).toBe("acme/new");
    expect(result.current.issuesRepo).toBe("acme/new");
  });

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

  it("clears the selected repository when the GitHub connection changes", () => {
    const connected = new Set<IntegrationId>(["github"]);
    const { result } = renderHook(() => useOnboardingSetupState("Payments", { connected, simulateDiscovery: false }));

    act(() => {
      result.current.selectVcsHost("github");
      result.current.selectRepo("acme/payments");
    });
    expect(result.current.selectedRepo).toBe("acme/payments");

    act(() => result.current.clearRepository());
    expect(result.current.selectedRepo).toBeNull();
    expect(result.current.repoReady).toBe(false);
  });

  it("marks the agent step ready when remaining credit is greater than zero", () => {
    const { result } = renderHook(() =>
      useOnboardingSetupState("Payments", {
        connected: new Set<IntegrationId>(),
        remainingCreditCents: 5000,
        simulateDiscovery: false,
      }),
    );

    expect(result.current.agentReady).toBe(true);
  });

  it("marks the agent step ready when OpenAI is connected and credit is empty", () => {
    const { result } = renderHook(() =>
      useOnboardingSetupState("Payments", {
        connected: new Set<IntegrationId>(["openai"]),
        remainingCreditCents: 0,
        simulateDiscovery: false,
      }),
    );

    expect(result.current.agentReady).toBe(true);
  });

  it("lets setup finish when OpenRouter is connected", () => {
    const connected = new Set<IntegrationId>(["github", "openrouter"]);
    const { result } = renderHook(() =>
      useOnboardingSetupState("Payments", {
        connected,
        remainingCreditCents: 0,
        simulateDiscovery: false,
      }),
    );

    act(() => {
      result.current.selectVcsHost("github");
      result.current.selectRepo("acme/payments");
    });

    expect(result.current.agentReady).toBe(true);
    expect(result.current.canFinish).toBe(true);
  });

  it("lets setup finish when Anthropic is connected or remaining credit is greater than zero", () => {
    const connected = new Set<IntegrationId>(["github", "claude"]);
    const { result } = renderHook(() =>
      useOnboardingSetupState("Payments", {
        connected,
        remainingCreditCents: 0,
        simulateDiscovery: false,
      }),
    );

    act(() => {
      result.current.selectVcsHost("github");
      result.current.selectRepo("acme/payments");
    });

    expect(result.current.agentReady).toBe(true);
    expect(result.current.canFinish).toBe(true);
  });
});
