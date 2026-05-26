import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { isChunkLoadError, loadModuleWithReload } from "@/lib/lazyWithReload";

const RELOAD_FLAG = "superplane:lazy-reload-attempted";

describe("isChunkLoadError", () => {
  it("recognizes the Chrome dynamic-import failure message", () => {
    expect(isChunkLoadError(new Error("Failed to fetch dynamically imported module: https://example.com/a.js"))).toBe(
      true,
    );
  });

  it("recognizes the Firefox dynamic-import failure message", () => {
    expect(isChunkLoadError(new Error("error loading dynamically imported module"))).toBe(true);
  });

  it("recognizes the Safari dynamic-import failure message", () => {
    expect(isChunkLoadError(new Error("Importing a module script failed."))).toBe(true);
  });

  it("recognizes the Vite CSS preload failure message", () => {
    expect(isChunkLoadError(new Error("Unable to preload CSS for https://example.com/a.css"))).toBe(true);
  });

  it("returns false for unrelated errors", () => {
    expect(isChunkLoadError(new Error("Boom"))).toBe(false);
    expect(isChunkLoadError(null)).toBe(false);
    expect(isChunkLoadError(undefined)).toBe(false);
  });
});

describe("loadModuleWithReload", () => {
  const reloadMock = vi.fn();

  beforeEach(() => {
    reloadMock.mockReset();
    window.sessionStorage.clear();
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { reload: reloadMock },
    });
  });

  afterEach(() => {
    window.sessionStorage.clear();
  });

  it("resolves the module on the first attempt when the factory succeeds", async () => {
    const factory = vi.fn(async () => ({ default: "ok" }));

    await expect(loadModuleWithReload(factory)).resolves.toEqual({ default: "ok" });
    expect(factory).toHaveBeenCalledTimes(1);
    expect(reloadMock).not.toHaveBeenCalled();
    expect(window.sessionStorage.getItem(RELOAD_FLAG)).toBeNull();
  });

  it("retries once and resolves when the first attempt fails with a chunk-load error", async () => {
    const factory = vi
      .fn<() => Promise<{ default: string }>>()
      .mockRejectedValueOnce(new Error("Failed to fetch dynamically imported module: https://example.com/a.js"))
      .mockResolvedValueOnce({ default: "ok" });

    await expect(loadModuleWithReload(factory, { retryDelayMs: 0 })).resolves.toEqual({ default: "ok" });
    expect(factory).toHaveBeenCalledTimes(2);
    expect(reloadMock).not.toHaveBeenCalled();
  });

  it("reloads the page when both attempts fail with a chunk-load error and returns a never-resolving promise", async () => {
    const factory = vi
      .fn<() => Promise<{ default: string }>>()
      .mockRejectedValue(new Error("Failed to fetch dynamically imported module: https://example.com/a.js"));

    const pending = loadModuleWithReload(factory, { retryDelayMs: 0 });

    const settled = await Promise.race([
      pending.then(
        () => "resolved",
        () => "rejected",
      ),
      new Promise((resolve) => setTimeout(() => resolve("still-pending"), 25)),
    ]);

    expect(settled).toBe("still-pending");
    expect(factory).toHaveBeenCalledTimes(2);
    expect(reloadMock).toHaveBeenCalledTimes(1);
    expect(window.sessionStorage.getItem(RELOAD_FLAG)).toBe("true");
  });

  it("does not reload again if the session already attempted a reload", async () => {
    window.sessionStorage.setItem(RELOAD_FLAG, "true");

    const chunkError = new Error("Failed to fetch dynamically imported module: https://example.com/a.js");
    const factory = vi.fn<() => Promise<{ default: string }>>().mockRejectedValue(chunkError);

    await expect(loadModuleWithReload(factory, { retryDelayMs: 0 })).rejects.toThrow(chunkError);
    expect(factory).toHaveBeenCalledTimes(2);
    expect(reloadMock).not.toHaveBeenCalled();
  });

  it("does not retry or reload for non-chunk-load errors", async () => {
    const unrelated = new Error("Boom");
    const factory = vi.fn<() => Promise<{ default: string }>>().mockRejectedValue(unrelated);

    await expect(loadModuleWithReload(factory, { retryDelayMs: 0 })).rejects.toThrow(unrelated);
    expect(factory).toHaveBeenCalledTimes(1);
    expect(reloadMock).not.toHaveBeenCalled();
  });

  it("clears the reload flag once the import eventually succeeds", async () => {
    window.sessionStorage.setItem(RELOAD_FLAG, "true");

    const factory = vi.fn(async () => ({ default: "ok" }));

    await loadModuleWithReload(factory);

    expect(window.sessionStorage.getItem(RELOAD_FLAG)).toBeNull();
  });
});
