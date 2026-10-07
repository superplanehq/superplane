import { describe, expect, it } from "bun:test";

import { createLocalMonacoInit, type Cancelable } from "./localMonacoInit";

function resolvedEditor(value: string, onCancel?: () => void): Cancelable<string> {
  const result = Promise.resolve(value) as Cancelable<string>;
  result.cancel = () => {
    onCancel?.();
  };
  return result;
}

describe("createLocalMonacoInit", () => {
  it("does not start the editor loader until the local editor is ready", async () => {
    let releaseLocalMonaco = () => {};
    let initCalls = 0;
    const start = createLocalMonacoInit(
      () => {
        initCalls += 1;
        return resolvedEditor("local");
      },
      () =>
        new Promise<void>((resolve) => {
          releaseLocalMonaco = resolve;
        }),
    );

    const pending = start();
    await Promise.resolve();
    expect(initCalls).toBe(0);

    releaseLocalMonaco();
    await expect(pending).resolves.toBe("local");
    expect(initCalls).toBe(1);
  });

  it("loads the local editor once when editors overlap", async () => {
    let loads = 0;
    let releaseLocalMonaco = () => {};
    const start = createLocalMonacoInit(
      () => resolvedEditor("ok"),
      () => {
        loads += 1;
        return new Promise<void>((resolve) => {
          releaseLocalMonaco = resolve;
        });
      },
    );

    const first = start();
    const second = start();
    expect(loads).toBe(1);

    releaseLocalMonaco();
    await expect(Promise.all([first, second])).resolves.toEqual(["ok", "ok"]);
    expect(loads).toBe(1);
  });

  it("does not start the editor loader when the local editor fails", async () => {
    let initCalls = 0;
    const start = createLocalMonacoInit(
      () => {
        initCalls += 1;
        return resolvedEditor("cdn");
      },
      () => Promise.reject(new Error("missing editor")),
    );

    await expect(start()).rejects.toThrow("missing editor");
    expect(initCalls).toBe(0);
  });

  it("tries the local editor again after a failed load", async () => {
    let loads = 0;
    const start = createLocalMonacoInit(
      () => resolvedEditor("ok"),
      () => {
        loads += 1;
        if (loads === 1) {
          return Promise.reject(new Error("missing editor"));
        }
        return Promise.resolve();
      },
    );

    await expect(start()).rejects.toThrow("missing editor");
    await expect(start()).resolves.toBe("ok");
    expect(loads).toBe(2);
  });

  it("cancels one editor without starting the loader for that call", async () => {
    let releaseLocalMonaco = () => {};
    let initCalls = 0;
    const start = createLocalMonacoInit(
      () => {
        initCalls += 1;
        return resolvedEditor("ok");
      },
      () =>
        new Promise<void>((resolve) => {
          releaseLocalMonaco = resolve;
        }),
    );

    const pending = start();
    pending.cancel();
    releaseLocalMonaco();

    await expect(pending).rejects.toEqual({ type: "cancelation" });
    expect(initCalls).toBe(0);
  });

  it("forwards cancel to the loader after the local editor is ready", async () => {
    let cancelled = false;
    const start = createLocalMonacoInit(
      () =>
        resolvedEditor("ok", () => {
          cancelled = true;
        }),
      () => Promise.resolve(),
    );

    const pending = start();
    await pending;
    pending.cancel();

    expect(cancelled).toBe(true);
  });
});
