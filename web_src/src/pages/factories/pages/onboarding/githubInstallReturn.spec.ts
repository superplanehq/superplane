import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import {
  clearGitHubInstallStarted,
  githubAccessKeys,
  markGitHubInstallStarted,
  useGitHubInstallReturn,
} from "./githubInstallReturn";

const scope = { accountId: "account-1", factoryId: "factory-1" };

describe("useGitHubInstallReturn", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("does not check when the user did not go to GitHub", () => {
    const { result } = renderHook(() => useGitHubInstallReturn(scope, ["installation:101"], vi.fn()));

    expect(result.current).toBe(false);
  });

  it("checks after a return from GitHub until new access arrives", () => {
    markGitHubInstallStarted(scope, ["installation:101"]);
    const { result, rerender } = renderHook(({ keys }) => useGitHubInstallReturn(scope, keys, vi.fn()), {
      initialProps: { keys: ["installation:101"] as string[] | undefined },
    });
    expect(result.current).toBe(true);

    rerender({ keys: ["installation:101", "request:7"] });

    expect(result.current).toBe(false);
    const again = renderHook(() => useGitHubInstallReturn(scope, ["installation:101", "request:7"], vi.fn()));
    expect(again.result.current).toBe(false);
  });

  it("does not check for another person or workspace", () => {
    markGitHubInstallStarted(scope, []);

    const otherPerson = renderHook(() =>
      useGitHubInstallReturn({ accountId: "account-2", factoryId: "factory-1" }, [], vi.fn()),
    );
    const otherWorkspace = renderHook(() =>
      useGitHubInstallReturn({ accountId: "account-1", factoryId: "factory-2" }, [], vi.fn()),
    );

    expect(otherPerson.result.current).toBe(false);
    expect(otherWorkspace.result.current).toBe(false);
  });

  it("does not check after the GitHub launch fails", () => {
    markGitHubInstallStarted(scope, []);
    clearGitHubInstallStarted(scope);

    const { result } = renderHook(() => useGitHubInstallReturn(scope, [], vi.fn()));

    expect(result.current).toBe(false);
  });
});

describe("githubAccessKeys", () => {
  it("lists installations, repositories, and install requests once", () => {
    expect(
      githubAccessKeys({
        repositories: [
          { installationId: "101", repositoryId: "201", fullName: "acme/api" },
          { installationId: "101", repositoryId: "202", fullName: "acme/web" },
        ],
        pendingRequests: [{ requestId: "7", accountLogin: "octo" }],
      }),
    ).toEqual(["installation:101", "repository:201", "repository:202", "request:7"]);
  });
});
