import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useSearchParams } from "react-router-dom";
import { beforeEach, describe, expect, it } from "vitest";

import { useConsoleActivePageInitial, useConsoleActivePageSync } from "./useConsoleActivePage";

/**
 * Component-based harness for the initial + sync hook pair, mirroring
 * the way `ConsoleOverlay` wires them together. Using `render` plus
 * DOM assertions (rather than `renderHook`) keeps the test transparent
 * about which React scheduler ticks we are observing — that visibility
 * mattered when tracking down a URL <-> state reconciliation loop.
 */
function Harness({
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

  return (
    <div>
      <div data-testid="active">{activePageId ?? "null"}</div>
      <div data-testid="url">{searchParams.get("page") ?? "null"}</div>
      <button
        onClick={() =>
          setSearchParams(
            (prev) => {
              const next = new URLSearchParams(prev);
              next.set("page", "details");
              return next;
            },
            { replace: true },
          )
        }
      >
        go-details-via-url
      </button>
      <button onClick={() => setActivePageId("details")}>click-details</button>
      <button
        onClick={() =>
          setSearchParams(
            (prev) => {
              const next = new URLSearchParams(prev);
              next.set("page", "ghost");
              return next;
            },
            { replace: true },
          )
        }
      >
        go-ghost-via-url
      </button>
    </div>
  );
}

function renderHarness(initialSearch: string, livePageIds: string[]) {
  return render(
    <MemoryRouter initialEntries={[{ pathname: "/", search: initialSearch }]}>
      <Harness canvasId="canvas" persistedPageIds={["overview", "details"]} livePageIds={livePageIds} />
    </MemoryRouter>,
  );
}

describe("useConsoleActivePage — URL / state sync", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("renders the initial active page from the URL", () => {
    renderHarness("?page=overview", ["overview", "details"]);
    expect(screen.getByTestId("active").textContent).toBe("overview");
    expect(screen.getByTestId("url").textContent).toBe("overview");
  });

  it("switches active page when the URL param changes to a valid live page", async () => {
    // Simulates back/forward navigation or a deep link swap. The
    // previous active id is still valid, but the URL now names a
    // different valid page — the grid must follow instead of ignoring
    // the new param because the current id is still technically OK.
    const user = userEvent.setup();
    renderHarness("?page=overview", ["overview", "details"]);

    await user.click(screen.getByText("go-details-via-url"));

    await waitFor(() => {
      expect(screen.getByTestId("active").textContent).toBe("details");
    });
    expect(screen.getByTestId("url").textContent).toBe("details");
  });

  it("projects a click into the URL without reverting", async () => {
    // The user clicks a tab (in-app state change) while the URL still
    // names the previous tab. State moves first, then the sync effect
    // must write the URL to match — without an adoption pass fighting
    // it and looping back to the prior tab.
    const user = userEvent.setup();
    renderHarness("?page=overview", ["overview", "details"]);

    await user.click(screen.getByText("click-details"));

    await waitFor(() => {
      expect(screen.getByTestId("url").textContent).toBe("details");
    });
    expect(screen.getByTestId("active").textContent).toBe("details");
  });

  it("adopts a URL page once pages hydrate asynchronously", async () => {
    // The console query is empty on first render (still loading) so
    // `useConsoleActivePageInitial` returns null. When the pages
    // arrive on a later render, the URL param never *changed* so a
    // change-only adoption path would miss it. Case 2a must catch
    // this and resolve against the freshly-loaded live list.
    const { rerender } = render(
      <MemoryRouter initialEntries={[{ pathname: "/", search: "?page=details" }]}>
        <Harness canvasId="canvas" persistedPageIds={[]} livePageIds={[]} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("active").textContent).toBe("null");

    rerender(
      <MemoryRouter initialEntries={[{ pathname: "/", search: "?page=details" }]}>
        <Harness canvasId="canvas" persistedPageIds={["overview", "details"]} livePageIds={["overview", "details"]} />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getByTestId("active").textContent).toBe("details");
    });
  });

  it("adopts a URL page from persisted ids before live state hydrates", async () => {
    // The console query settles a tick before `useConsolePagesState`
    // mirrors the committed pages into local state. In that window
    // `livePageIds` is still empty but `persistedPageIds` is not. The
    // sync effect must fall back to `persistedPageIds` so the multi-
    // page URL is honored on the first render where the query has
    // data, without waiting for a second commit.
    const { rerender } = render(
      <MemoryRouter initialEntries={[{ pathname: "/", search: "?page=details" }]}>
        <Harness canvasId="canvas" persistedPageIds={[]} livePageIds={[]} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("active").textContent).toBe("null");

    // Persisted arrives; live is still empty (state not yet hydrated).
    rerender(
      <MemoryRouter initialEntries={[{ pathname: "/", search: "?page=details" }]}>
        <Harness canvasId="canvas" persistedPageIds={["overview", "details"]} livePageIds={[]} />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getByTestId("active").textContent).toBe("details");
    });
  });

  it("falls back and rewrites the URL when the param points to a removed page", async () => {
    // `ghost` is not in the live list, so the resolver falls back to
    // the first available id and the sync effect rewrites the stale
    // param so the URL matches the rendered tab.
    const user = userEvent.setup();
    renderHarness("?page=overview", ["overview", "details"]);

    await user.click(screen.getByText("go-ghost-via-url"));

    await waitFor(() => {
      expect(screen.getByTestId("url").textContent).toBe("overview");
    });
    expect(screen.getByTestId("active").textContent).toBe("overview");
  });
});
