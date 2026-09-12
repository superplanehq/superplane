import { createElement, type ReactNode } from "react";
import { vi } from "bun:test";

import "@testing-library/jest-dom/vitest";

// Happy DOM has no CSS layout. Recharts ResponsiveContainer then reads
// getBoundingClientRect after mount, overwrites initialDimension with 0x0,
// and warns that the chart width and height must be greater than 0.
// A numeric size skips that measure so every chart test stays quiet.
vi.mock("recharts", () => ({
  ResponsiveContainer: ({ children }: { children: ReactNode }) =>
    createElement("div", { style: { width: 760, height: 240 } }, children),
}));

// jsdom doesn't ship ResizeObserver; several UI primitives depend on it.
// Provide a no-op so every test file gets it for free instead of having to
// declare it locally.
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

// Happy DOM's WebSocket connects for real. A failed handshake emits an
// unhandled ErrorEvent that Bun treats as a test failure, often on the
// next case in the file. Tests never need a live socket.
class SilentWebSocket {
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSING = 2;
  static readonly CLOSED = 3;

  readonly CONNECTING = SilentWebSocket.CONNECTING;
  readonly OPEN = SilentWebSocket.OPEN;
  readonly CLOSING = SilentWebSocket.CLOSING;
  readonly CLOSED = SilentWebSocket.CLOSED;
  readonly url: string;
  readyState = SilentWebSocket.CONNECTING;
  protocol = "";
  extensions = "";
  bufferedAmount = 0;
  binaryType: BinaryType = "blob";
  onopen: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;

  constructor(url: string | URL) {
    this.url = String(url);
  }

  close() {
    this.readyState = SilentWebSocket.CLOSED;
  }

  send() {}
  addEventListener() {}
  removeEventListener() {}
  dispatchEvent() {
    return false;
  }
}

Object.defineProperty(globalThis, "WebSocket", {
  configurable: true,
  writable: true,
  value: SilentWebSocket,
});
Object.defineProperty(window, "WebSocket", {
  configurable: true,
  writable: true,
  value: SilentWebSocket,
});

Object.defineProperty(HTMLCanvasElement.prototype, "getContext", {
  configurable: true,
  writable: true,
  value: () => ({
    font: "",
    measureText: (text: string) => ({ width: text.length * 7 }),
  }),
});
