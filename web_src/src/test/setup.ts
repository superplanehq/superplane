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

// jsdom does not implement DOMMatrixReadOnly. React Flow reads viewport zoom
// with `new DOMMatrixReadOnly(style.transform)` when node internals update.
if (typeof window.DOMMatrixReadOnly !== "function") {
  class DOMMatrixReadOnlyStub {
    m11 = 1;
    m12 = 0;
    m21 = 0;
    m22 = 1;
    m41 = 0;
    m42 = 0;

    constructor(init?: string) {
      const values = parseCssMatrix2d(init);
      if (!values) {
        return;
      }
      const [a, b, c, d, e, f] = values;
      this.m11 = a;
      this.m12 = b;
      this.m21 = c;
      this.m22 = d;
      this.m41 = e;
      this.m42 = f;
    }
  }

  Object.defineProperty(window, "DOMMatrixReadOnly", {
    configurable: true,
    writable: true,
    value: DOMMatrixReadOnlyStub,
  });
}

function parseCssMatrix2d(init: string | undefined): number[] | undefined {
  if (!init || init === "none") {
    return undefined;
  }
  const match = /matrix\(([^)]+)\)/.exec(init);
  if (!match) {
    return undefined;
  }
  const values = match[1].split(",").map((part) => Number(part.trim()));
  if (values.length < 6 || values.some((value) => Number.isNaN(value))) {
    return undefined;
  }
  return values;
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
