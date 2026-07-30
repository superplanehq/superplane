import { afterEach } from "vitest";

import "@testing-library/jest-dom";

// Drain `setTimeout(0)` callbacks queued by libraries like Radix UI's
// `react-focus-scope` on unmount. Radix schedules a synthetic focus
// dispatch 0 ms after the component unmounts; on a busy CI worker the
// timer can fire AFTER Vitest tears down the current jsdom realm,
// which then makes `new CustomEvent(...)` return `undefined` and
// `container.dispatchEvent(undefined)` throw:
//
//   TypeError: Failed to execute 'dispatchEvent' on 'EventTarget':
//   parameter 1 is not of type 'Event'.
//     at Timeout._onTimeout @radix-ui/react-focus-scope/…:92
//
// Vitest reports that as an unhandled error and fails the whole shard
// even though every test in it passed (observed intermittently on
// `check.test.ui.shard`, e.g. shard 3/4 of PR #6383 reporting
// "1044 passed / 1 error"). Yielding one macrotask in `afterEach`
// lets the queued dispatch fire while jsdom is still healthy. The
// cost is a single event-loop tick per test, negligible next to
// per-test render/mount overhead.
afterEach(async () => {
  await new Promise<void>((resolve) => setTimeout(resolve, 0));
});

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
