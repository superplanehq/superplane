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

// jsdom doesn't implement matchMedia; ThemeProvider reads it to resolve
// "system" theme preference. Tests can override this per file when needed.
if (typeof window.matchMedia === "undefined") {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener() {},
      removeListener() {},
      addEventListener() {},
      removeEventListener() {},
      dispatchEvent: () => false,
    }),
  });
}

Object.defineProperty(HTMLCanvasElement.prototype, "getContext", {
  configurable: true,
  writable: true,
  value: () => ({
    font: "",
    measureText: (text: string) => ({ width: text.length * 7 }),
  }),
});
