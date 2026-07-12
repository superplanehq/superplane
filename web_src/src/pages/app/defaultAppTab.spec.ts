import { describe, expect, it } from "vitest";

import {
  buildAppTabSearchParams,
  resolveDefaultTab,
  urlDeepLinksWithoutTabPick,
  urlPinsNavigation,
  urlViewFlagsToTab,
  type ConsoleQueryLike,
  type UrlViewFlags,
} from "./defaultAppTab";

const CANVAS_FLAGS: UrlViewFlags = {
  isRunInspectionMode: false,
  isMemoryMode: false,
  isFilesMode: false,
  isConsoleMode: false,
};

const CONSOLE_FLAGS: UrlViewFlags = { ...CANVAS_FLAGS, isConsoleMode: true };
const MEMORY_FLAGS: UrlViewFlags = { ...CANVAS_FLAGS, isMemoryMode: true };
const FILES_FLAGS: UrlViewFlags = { ...CANVAS_FLAGS, isFilesMode: true };
const RUN_FLAGS: UrlViewFlags = { ...CANVAS_FLAGS, isRunInspectionMode: true };

function consoleLoaded(panelCount: number): ConsoleQueryLike {
  return {
    isSuccess: true,
    isError: false,
    data: { panels: Array.from({ length: panelCount }, () => ({})) },
  };
}

const consoleLoading: ConsoleQueryLike = { isSuccess: false, isError: false, data: undefined };
const consoleErrored: ConsoleQueryLike = { isSuccess: false, isError: true, data: undefined };

describe("urlViewFlagsToTab", () => {
  it("returns null while run inspection is active — run maps to no tab", () => {
    expect(urlViewFlagsToTab(RUN_FLAGS)).toBeNull();
  });

  it("maps the URL view flags to the matching tab", () => {
    expect(urlViewFlagsToTab(CANVAS_FLAGS)).toBe("canvas");
    expect(urlViewFlagsToTab(CONSOLE_FLAGS)).toBe("console");
    expect(urlViewFlagsToTab(MEMORY_FLAGS)).toBe("memory");
    expect(urlViewFlagsToTab(FILES_FLAGS)).toBe("files");
  });
});

describe("urlPinsNavigation", () => {
  it("returns false for a bare canvas URL", () => {
    expect(urlPinsNavigation(new URLSearchParams())).toBe(false);
  });

  it.each(["console", "dashboard", "memory", "files"])(
    "pins on tab-selecting view=%s (dashboard is the legacy Console alias)",
    (view) => {
      expect(urlPinsNavigation(new URLSearchParams(`view=${view}`))).toBe(true);
    },
  );

  it.each(["runs", "versions"])(
    "does not pin on legacy view=%s — those values select no tab and are cleaned up on mount",
    (view) => {
      expect(urlPinsNavigation(new URLSearchParams(`view=${view}`))).toBe(false);
    },
  );

  it.each(["run", "version", "edit", "sidebar", "node", "file"])(
    "pins on deep-link param %s so the redirect never strips it",
    (param) => {
      expect(urlPinsNavigation(new URLSearchParams(`${param}=x`))).toBe(true);
    },
  );

  it("ignores empty deep-link params", () => {
    expect(urlPinsNavigation(new URLSearchParams("run="))).toBe(false);
  });
});

describe("urlDeepLinksWithoutTabPick", () => {
  it("returns true for a bare deep link (no view)", () => {
    expect(urlDeepLinksWithoutTabPick(new URLSearchParams("node=n1&sidebar=1"))).toBe(true);
  });

  it("returns false when the deep link is paired with an explicit tab pick", () => {
    expect(urlDeepLinksWithoutTabPick(new URLSearchParams("view=memory&node=n1"))).toBe(false);
  });

  it("returns false for run-only URLs — run has its own persistence guard", () => {
    expect(urlDeepLinksWithoutTabPick(new URLSearchParams("run=r1"))).toBe(false);
  });
});

describe("buildAppTabSearchParams", () => {
  it("clears view when landing on Canvas", () => {
    const next = buildAppTabSearchParams("canvas", new URLSearchParams("view=console"));
    expect(next.get("view")).toBeNull();
  });

  it("sets view for a non-canvas tab", () => {
    const next = buildAppTabSearchParams("console", new URLSearchParams());
    expect(next.get("view")).toBe("console");
  });

  it("strips selection params that only make sense on the previous tab", () => {
    const next = buildAppTabSearchParams(
      "console",
      new URLSearchParams("run=r1&sidebar=1&node=n1&file=components/app.yaml"),
    );
    expect(next.get("run")).toBeNull();
    expect(next.get("sidebar")).toBeNull();
    expect(next.get("node")).toBeNull();
    expect(next.get("file")).toBeNull();
  });

  it("preserves unrelated params", () => {
    const next = buildAppTabSearchParams("memory", new URLSearchParams("foo=bar&view=console"));
    expect(next.get("foo")).toBe("bar");
    expect(next.get("view")).toBe("memory");
  });
});

describe("resolveDefaultTab", () => {
  it("prefers the stored tab even when the live console has panels", () => {
    const resolution = resolveDefaultTab({
      storedTab: "memory",
      liveConsoleQuery: consoleLoaded(3),
    });
    expect(resolution).toEqual({ settled: true, redirectTo: "memory" });
  });

  it("stays unsettled while the console query is still loading — waiting prevents locking in Canvas", () => {
    const resolution = resolveDefaultTab({
      storedTab: null,
      liveConsoleQuery: consoleLoading,
    });
    expect(resolution).toEqual({ settled: false });
  });

  it("redirects to Console when the live app has panels", () => {
    const resolution = resolveDefaultTab({
      storedTab: null,
      liveConsoleQuery: consoleLoaded(1),
    });
    expect(resolution).toEqual({ settled: true, redirectTo: "console" });
  });

  it("stays on Canvas when the live app has no panels", () => {
    const resolution = resolveDefaultTab({
      storedTab: null,
      liveConsoleQuery: consoleLoaded(0),
    });
    expect(resolution).toEqual({ settled: true, redirectTo: null });
  });

  it("settles on Canvas when the console read errored, so the gate does not spin forever", () => {
    const resolution = resolveDefaultTab({
      storedTab: null,
      liveConsoleQuery: consoleErrored,
    });
    expect(resolution).toEqual({ settled: true, redirectTo: null });
  });

  it("ignores an invalid stored tab and falls through to the live-console fallback", () => {
    const resolution = resolveDefaultTab({
      storedTab: "bogus" as never,
      liveConsoleQuery: consoleLoaded(1),
    });
    expect(resolution).toEqual({ settled: true, redirectTo: "console" });
  });
});
