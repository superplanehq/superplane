import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

const { resolveGithubDefaultBranch, showErrorToast } = vi.hoisted(() => ({
  resolveGithubDefaultBranch: vi.fn(),
  showErrorToast: vi.fn(),
}));

vi.mock("@/hooks/useIntegrations", () => ({
  resolveGithubDefaultBranch,
  useIntegrationResources: () => ({ data: [{ name: "main" }, { name: "develop" }] }),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast,
  showSuccessToast: vi.fn(),
}));

vi.mock("./useFactoryRepository", () => ({
  useFactoryRepository: () => ({ isPending: false, mutateAsync: vi.fn() }),
}));

import { useFactoryRepositoryDraft } from "./useFactoryRepositoryDraft";

type CatalogRepository = {
  fullName?: string | null;
  defaultBranch?: string | null;
};

describe("useFactoryRepositoryDraft", () => {
  beforeEach(() => {
    resolveGithubDefaultBranch.mockReset();
    showErrorToast.mockReset();
  });

  it("keeps the latest repository branch when an older lookup finishes last", async () => {
    const lookups = new Map<string, Deferred<string>>();
    resolveGithubDefaultBranch.mockImplementation(
      (_organizationId: string, _integrationId: string, repository: string) => {
        const pending = deferred<string>();
        lookups.set(repository, pending);
        return pending.promise;
      },
    );
    const { result } = renderHook(() => useFactoryRepositoryDraft(draftArgs()));

    act(() => result.current.selectRepository("acme/a"));
    act(() => result.current.selectRepository("acme/b"));

    await act(async () => {
      lookups.get("acme/a")?.resolve("main");
    });
    expect(result.current.branch).toBe("trunk");
    expect(result.current.isSaving).toBe(true);

    await act(async () => {
      lookups.get("acme/b")?.resolve("develop");
    });
    expect(result.current.branch).toBe("develop");
    expect(result.current.isSaving).toBe(false);
  });

  it("keeps an edited branch when repository choices refresh", () => {
    const { result, rerender } = renderHook(
      (catalogRepositories: CatalogRepository[]) =>
        useFactoryRepositoryDraft(
          draftArgs({
            savedBranch: "main",
            catalogRepositories,
          }),
        ),
      { initialProps: [{ fullName: "acme/api", defaultBranch: "main" }] },
    );

    act(() => result.current.setBranch("develop"));
    rerender([{ fullName: "acme/api", defaultBranch: "main" }]);

    expect(result.current.branch).toBe("develop");
    expect(result.current.isDirty).toBe(true);
  });

  it("resets the branch when the saved branch changes", () => {
    const { result, rerender } = renderHook(
      (savedBranch: string) => useFactoryRepositoryDraft(draftArgs({ savedBranch })),
      { initialProps: "main" },
    );

    act(() => result.current.setBranch("develop"));
    rerender("release");

    expect(result.current.branch).toBe("release");
    expect(result.current.isDirty).toBe(false);
  });

  it("clears the previous branch when the selected repository lookup fails", async () => {
    resolveGithubDefaultBranch.mockRejectedValue(new Error("network down"));
    const { result } = renderHook(() => useFactoryRepositoryDraft(draftArgs({ savedBranch: "main" })));

    act(() => result.current.selectRepository("acme/web"));

    await waitFor(() => expect(result.current.branch).toBe(""));
    expect(result.current.isSaving).toBe(false);
    expect(showErrorToast).toHaveBeenCalledWith("network down");
  });

  it("shows an error when the saved default branch cannot be loaded", async () => {
    resolveGithubDefaultBranch.mockRejectedValue(new Error("network down"));
    const { result } = renderHook(() => useFactoryRepositoryDraft(draftArgs({ savedBranch: "" })));

    await waitFor(() => expect(showErrorToast).toHaveBeenCalledWith("network down"));
    expect(result.current.branch).toBe("");
    expect(result.current.isSaving).toBe(false);
  });

  it("ignores a failed lookup after the user selects another repository", async () => {
    const lookups = new Map<string, Deferred<string>>();
    resolveGithubDefaultBranch.mockImplementation(
      (_organizationId: string, _integrationId: string, repository: string) => {
        const pending = deferred<string>();
        lookups.set(repository, pending);
        return pending.promise;
      },
    );
    const { result } = renderHook(() => useFactoryRepositoryDraft(draftArgs()));

    act(() => result.current.selectRepository("acme/a"));
    act(() => result.current.selectRepository("acme/b"));
    await act(async () => {
      lookups.get("acme/a")?.reject(new Error("network down"));
    });

    expect(showErrorToast).not.toHaveBeenCalled();
    expect(result.current.isSaving).toBe(true);

    await act(async () => {
      lookups.get("acme/b")?.resolve("develop");
    });
    expect(result.current.branch).toBe("develop");
    expect(showErrorToast).not.toHaveBeenCalled();
  });
});

function draftArgs(overrides?: {
  savedRepository?: string;
  savedBranch?: string;
  catalogRepositories?: CatalogRepository[];
}) {
  return {
    organizationId: "org-1",
    factoryId: "factory-1",
    factory: {
      onboarding: {
        appRepository: overrides?.savedRepository ?? "acme/api",
        defaultBranch: overrides && "savedBranch" in overrides ? overrides.savedBranch : "trunk",
      },
    },
    integrationId: "integration-1",
    catalogRepositories: overrides?.catalogRepositories,
  };
}

type Deferred<T> = {
  promise: Promise<T>;
  resolve: (value: T) => void;
  reject: (error: unknown) => void;
};

function deferred<T>(): Deferred<T> {
  let resolve: (value: T) => void = () => {};
  let reject: (error: unknown) => void = () => {};
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}
