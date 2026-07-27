import { act, renderHook } from "@testing-library/react";
import { MemoryRouter, useSearchParams } from "react-router-dom";
import { beforeEach, describe, expect, it } from "vitest";

import { useConsoleActivePageInitial, useConsoleActivePageSync } from "./useConsoleActivePage";

/**
 * Small wrapper that spins up the initial + sync hook pair the same
 * way `ConsoleOverlay` does, but exposes both `activePageId` and the
 * search params so tests can assert URL round-trips.
 */
function useSubject({
  canvasId,
  persistedPageIds,
  livePageIds,
}: {
  canvasId: string;
  persistedPageIds: string[];
  livePageIds: string[];
}) {
  const [searchParams, setSearchParams] = useSearchParams();
  const { activePageId, setActivePageId, rawPageParam } = useConsoleActivePageInitial({
    canvasId,
    persistedPageIds,
  });
  useConsoleActivePageSync({
    canvasId,
    livePageIds,
    activePageId,
    setActivePageId,
    rawPageParam,
    persistedPageIds,
  });
  return {
    activePageId,
    setActivePageId,
    pageParam: searchParams.get("page"),
    goToUrl: (search: string) => setSearchParams(new URLSearchParams(search), { replace: true }),
  };
}

function wrapper({ initialSearch, children }: { initialSearch: string; children: React.ReactNode }) {
  return <MemoryRouter initialEntries={[{ pathname: "/", search: initialSearch }]}>{children}</MemoryRouter>;
}

describe("useConsoleActivePage — URL / state sync", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("switches active page when the URL param changes to a valid live page", () => {
    // Simulates back/forward navigation or a deep link swap. The
    // previous active id is still valid in the live page list, but the
    // URL now names a different valid page — the grid must follow.
    const initial = { canvasId: "canvas", persistedPageIds: ["overview", "details"], livePageIds: ["overview", "details"] };
    const { result, rerender } = renderHook(({ props, search }) => useSubject(props), {
      initialProps: { props: initial, search: "?page=overview" },
      wrapper: ({ children }) => wrapper({ initialSearch: "?page=overview", children }),
    });

    expect(result.current.activePageId).toBe("overview");

    act(() => {
      result.current.goToUrl("?page=details");
    });
    rerender({ props: initial, search: "?page=details" });

    expect(result.current.activePageId).toBe("details");
    expect(result.current.pageParam).toBe("details");
  });

  it("does not clobber a click when the URL param is still stale", () => {
    // The user clicks tab B while the URL still names A. State should
    // move to B, then the URL sync effect updates the URL — the
    // re-resolution effect must not revert the click back to A.
    const initial = { canvasId: "canvas", persistedPageIds: ["a", "b"], livePageIds: ["a", "b"] };
    const { result, rerender } = renderHook(({ props }) => useSubject(props), {
      initialProps: { props: initial },
      wrapper: ({ children }) => wrapper({ initialSearch: "?page=a", children }),
    });

    expect(result.current.activePageId).toBe("a");

    act(() => {
      result.current.setActivePageId("b");
    });
    rerender({ props: initial });

    expect(result.current.activePageId).toBe("b");
    expect(result.current.pageParam).toBe("b");
  });

  it("falls back to the first live page when the URL points to a removed page", () => {
    const initial = { canvasId: "canvas", persistedPageIds: ["a", "b"], livePageIds: ["a", "b"] };
    const { result, rerender } = renderHook(({ props }) => useSubject(props), {
      initialProps: { props: initial },
      wrapper: ({ children }) => wrapper({ initialSearch: "?page=ghost", children }),
    });

    // `ghost` is not in the live list — the resolver falls back to the
    // first available id (`a`).
    expect(result.current.activePageId).toBe("a");
    rerender({ props: initial });
    // The URL sync effect also clears the invalid param out.
    expect(result.current.pageParam).toBe("a");
  });
});
