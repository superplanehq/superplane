import { describe, expect, it, vi } from "bun:test";

const localMonaco = { editor: {} };
const configureLoader = vi.fn();

vi.mock("monaco-editor", () => localMonaco);
vi.mock("@monaco-editor/react", () => ({ loader: { config: configureLoader } }));

const { getMonacoWorkerType, setupMonaco } = await import("./setupMonaco");

describe("Monaco worker routing", () => {
  it.each([
    ["json", "json"],
    ["html", "html"],
    ["handlebars", "html"],
    ["razor", "html"],
    ["css", "css"],
    ["scss", "css"],
    ["less", "css"],
    ["javascript", "typescript"],
    ["typescript", "typescript"],
    ["yaml", "editor"],
    ["shell", "editor"],
    ["xml", "editor"],
    ["markdown", "editor"],
    ["plaintext", "editor"],
    ["", "editor"],
  ])("routes %s to the %s worker without constructing it", (label, workerLabel) => {
    expect(getMonacoWorkerType(label)).toBe(workerLabel);
  });
});

describe("Monaco loader setup", () => {
  it("configures the loader with the local package before setup completes", async () => {
    await setupMonaco();

    expect(configureLoader).toHaveBeenCalledWith({ monaco: localMonaco });
    expect(self.MonacoEnvironment?.getWorker).toBeDefined();
  });
});
