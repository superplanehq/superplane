import "@testing-library/jest-dom";

// jsdom doesn't ship ResizeObserver; several UI primitives (e.g. ModeToggle's
// sliding pill) depend on it. Provide a no-op so every test file gets it for
// free instead of having to declare it locally.
if (typeof globalThis.ResizeObserver === "undefined") {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
}
