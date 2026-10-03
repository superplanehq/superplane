import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { githubAccessKeys, markGitHubInstallStarted, useGitHubInstallReturn } from "./githubInstallReturn";

describe("useGitHubInstallReturn", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("does not check when the user did not go to GitHub", () => {
    const { result } = renderHook(() => useGitHubInstallReturn(["installation:101"], vi.fn()));

    expect(result.current).toBe(false);
  });

  it("checks after a return from GitHub until new access arrives", () => {
    markGitHubInstallStarted(["installation:101"]);
    const { result, rerender } = renderHook(({ keys }) => useGitHubInstallReturn(keys, vi.fn()), {
      initialProps: { keys: ["installation:101"] as string[] | undefined },
    });
    expect(result.current).toBe(true);

    rerender({ keys: ["installation:101", "request:7"] });

    expect(result.current).toBe(false);
    const again = renderHook(() => useGitHubInstallReturn(["installation:101", "request:7"], vi.fn()));
    expect(again.result.current).toBe(false);
  });
});

describe("githubAccessKeys", () => {
  it("lists each installation once and each install request", () => {
    expect(
      githubAccessKeys({
        repositories: [
          { installationId: "101", fullName: "acme/api" },
          { installationId: "101", fullName: "acme/web" },
        ],
        pendingRequests: [{ requestId: "7", accountLogin: "octo" }],
      }),
    ).toEqual(["installation:101", "request:7"]);
  });
});
