import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useIntegration, useIntegrationResources } from "@/hooks/useIntegrations";

import { useOnboardingGithubRepos } from "./useOnboardingGithubRepos";

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegration: vi.fn(),
  useIntegrationResources: vi.fn(),
}));

describe("useOnboardingGithubRepos", () => {
  beforeEach(() => {
    vi.mocked(useIntegration).mockReturnValue({ data: null } as ReturnType<typeof useIntegration>);
  });

  it("keeps repositories visible during a background refresh", () => {
    vi.mocked(useIntegrationResources).mockReturnValue({
      data: [{ id: "acme/api", name: "acme/api" }],
      error: null,
      isFetching: true,
      isPending: false,
    } as ReturnType<typeof useIntegrationResources>);

    const { result } = renderHook(() => useOnboardingGithubRepos("org-1", "github-1"));

    expect(result.current.repositories).toEqual(["acme/api"]);
    expect(result.current.repositoriesLoading).toBe(false);
  });

  it("shows the loader during the initial repository request", () => {
    vi.mocked(useIntegrationResources).mockReturnValue({
      data: undefined,
      error: null,
      isFetching: true,
      isPending: true,
    } as ReturnType<typeof useIntegrationResources>);

    const { result } = renderHook(() => useOnboardingGithubRepos("org-1", "github-1"));

    expect(result.current.repositories).toEqual([]);
    expect(result.current.repositoriesLoading).toBe(true);
  });
});
