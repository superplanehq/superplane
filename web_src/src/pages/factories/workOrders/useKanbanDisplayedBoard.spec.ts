import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { KANBAN_BOARD_MOTION_ROOT_CLASS, useKanbanDisplayedBoard, type KanbanCardPlacement } from "./kanbanCardMotion";

describe("useKanbanDisplayedBoard", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    Reflect.deleteProperty(document, "startViewTransition");
    Reflect.deleteProperty(document, "activeViewTransition");
    document.documentElement.classList.remove(KANBAN_BOARD_MOTION_ROOT_CLASS);
  });

  function stubKanbanViewTransition() {
    const startViewTransition = vi.fn((options: { types?: string[]; update?: () => void } | (() => void)) => {
      queueMicrotask(() => {
        if (typeof options === "function") {
          options();
          return;
        }
        options.update?.();
      });
      return {
        skipTransition: vi.fn(),
        finished: Promise.resolve(),
      };
    });
    vi.stubGlobal("matchMedia", (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }));
    Object.defineProperty(document, "startViewTransition", {
      configurable: true,
      value: startViewTransition,
    });
    Object.defineProperty(document, "activeViewTransition", {
      configurable: true,
      value: null,
      writable: true,
    });
    return startViewTransition;
  }

  it("commits the incoming snapshot immediately when motion cannot run", () => {
    const first: KanbanCardPlacement[] = [{ id: "a", column: "backlog" }];
    const second: KanbanCardPlacement[] = [{ id: "a", column: "phase-0" }];
    const { result, rerender } = renderHook(
      ({ value, placements }: { value: string; placements: KanbanCardPlacement[] }) =>
        useKanbanDisplayedBoard(value, placements),
      { initialProps: { value: "draft", placements: first } },
    );

    expect(result.current).toBe("draft");
    rerender({ value: "running", placements: second });
    expect(result.current).toBe("running");
  });

  it("returns the latest value when membership does not change", () => {
    const placements: KanbanCardPlacement[] = [{ id: "a", column: "backlog" }];
    const { result, rerender } = renderHook(
      ({ value }: { value: string }) => useKanbanDisplayedBoard(value, placements),
      { initialProps: { value: "title-1" } },
    );

    rerender({ value: "title-2" });
    expect(result.current).toBe("title-2");
  });

  it("starts a typed view transition when a card changes column", async () => {
    const startViewTransition = stubKanbanViewTransition();
    const first: KanbanCardPlacement[] = [{ id: "a", column: "backlog" }];
    const second: KanbanCardPlacement[] = [{ id: "a", column: "phase-0" }];
    const { rerender } = renderHook(
      ({ value, placements }: { value: string; placements: KanbanCardPlacement[] }) =>
        useKanbanDisplayedBoard(value, placements),
      { initialProps: { value: "draft", placements: first } },
    );

    rerender({ value: "running", placements: second });
    await act(async () => {
      await Promise.resolve();
    });

    expect(startViewTransition).toHaveBeenCalledTimes(1);
    expect(startViewTransition.mock.calls[0]?.[0]).toMatchObject({ types: ["kanban-board"] });
  });

  it("commits without a view transition when a task overlay is open", async () => {
    const startViewTransition = stubKanbanViewTransition();
    const first: KanbanCardPlacement[] = [{ id: "a", column: "backlog" }];
    const second: KanbanCardPlacement[] = [{ id: "a", column: "phase-0" }];
    const { result, rerender } = renderHook(
      ({
        value,
        placements,
        overlayOpen,
      }: {
        value: string;
        placements: KanbanCardPlacement[];
        overlayOpen: boolean;
      }) => useKanbanDisplayedBoard(value, placements, { overlayOpen }),
      { initialProps: { value: "draft", placements: first, overlayOpen: true } },
    );

    rerender({ value: "running", placements: second, overlayOpen: true });
    await act(async () => {
      await Promise.resolve();
    });

    expect(startViewTransition).not.toHaveBeenCalled();
    expect(result.current).toBe("running");
  });

  it("commits without a view transition when a modal dialog is in the document", async () => {
    const startViewTransition = stubKanbanViewTransition();
    const dialog = document.createElement("div");
    dialog.setAttribute("role", "dialog");
    dialog.setAttribute("aria-modal", "true");
    document.body.append(dialog);

    try {
      const first: KanbanCardPlacement[] = [{ id: "a", column: "backlog" }];
      const second: KanbanCardPlacement[] = [{ id: "a", column: "phase-0" }];
      const { result, rerender } = renderHook(
        ({ value, placements }: { value: string; placements: KanbanCardPlacement[] }) =>
          useKanbanDisplayedBoard(value, placements),
        { initialProps: { value: "draft", placements: first } },
      );

      rerender({ value: "running", placements: second });
      await act(async () => {
        await Promise.resolve();
      });

      expect(startViewTransition).not.toHaveBeenCalled();
      expect(result.current).toBe("running");
    } finally {
      dialog.remove();
    }
  });

  it("keeps motion styles when a newer transition replaces an older one", async () => {
    let resolveFirst = () => {};
    let resolveSecond = () => {};
    const firstFinished = new Promise<void>((resolve) => {
      resolveFirst = resolve;
    });
    const secondFinished = new Promise<void>((resolve) => {
      resolveSecond = resolve;
    });
    let startCount = 0;
    const startViewTransition = vi.fn((options: { types?: string[]; update?: () => void } | (() => void)) => {
      startCount += 1;
      queueMicrotask(() => {
        if (typeof options === "function") {
          options();
          return;
        }
        options.update?.();
      });
      return {
        skipTransition: vi.fn(),
        finished: startCount === 1 ? firstFinished : secondFinished,
      };
    });
    vi.stubGlobal("matchMedia", (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }));
    Object.defineProperty(document, "startViewTransition", {
      configurable: true,
      value: startViewTransition,
    });
    Object.defineProperty(document, "activeViewTransition", {
      configurable: true,
      value: null,
      writable: true,
    });

    const first: KanbanCardPlacement[] = [{ id: "a", column: "backlog" }];
    const second: KanbanCardPlacement[] = [{ id: "a", column: "phase-0" }];
    const third: KanbanCardPlacement[] = [{ id: "a", column: "verify" }];
    const { rerender } = renderHook(
      ({ value, placements }: { value: string; placements: KanbanCardPlacement[] }) =>
        useKanbanDisplayedBoard(value, placements),
      { initialProps: { value: "draft", placements: first } },
    );

    rerender({ value: "running", placements: second });
    await act(async () => {
      await Promise.resolve();
    });
    expect(document.documentElement.classList.contains(KANBAN_BOARD_MOTION_ROOT_CLASS)).toBe(true);

    rerender({ value: "verify", placements: third });
    await act(async () => {
      await Promise.resolve();
    });
    expect(document.documentElement.classList.contains(KANBAN_BOARD_MOTION_ROOT_CLASS)).toBe(true);

    await act(async () => {
      resolveFirst();
      await firstFinished;
    });
    expect(document.documentElement.classList.contains(KANBAN_BOARD_MOTION_ROOT_CLASS)).toBe(true);

    await act(async () => {
      resolveSecond();
      await secondFinished;
    });
    expect(document.documentElement.classList.contains(KANBAN_BOARD_MOTION_ROOT_CLASS)).toBe(false);
  });

  it("keeps motion styles when a content-only update arrives during a transition", async () => {
    let resolveFinished = () => {};
    const finished = new Promise<void>((resolve) => {
      resolveFinished = resolve;
    });
    const startViewTransition = vi.fn((options: { types?: string[]; update?: () => void } | (() => void)) => {
      queueMicrotask(() => {
        if (typeof options === "function") {
          options();
          return;
        }
        options.update?.();
      });
      return {
        skipTransition: vi.fn(),
        finished,
      };
    });
    vi.stubGlobal("matchMedia", (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeEventListener: vi.fn(),
      addEventListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }));
    Object.defineProperty(document, "startViewTransition", {
      configurable: true,
      value: startViewTransition,
    });
    Object.defineProperty(document, "activeViewTransition", {
      configurable: true,
      value: null,
      writable: true,
    });

    const first: KanbanCardPlacement[] = [{ id: "a", column: "backlog" }];
    const second: KanbanCardPlacement[] = [{ id: "a", column: "phase-0" }];
    const { result, rerender } = renderHook(
      ({ value, placements }: { value: string; placements: KanbanCardPlacement[] }) =>
        useKanbanDisplayedBoard(value, placements),
      { initialProps: { value: "draft", placements: first } },
    );

    rerender({ value: "running", placements: second });
    await act(async () => {
      await Promise.resolve();
    });
    expect(document.documentElement.classList.contains(KANBAN_BOARD_MOTION_ROOT_CLASS)).toBe(true);

    rerender({ value: "running, checks updated", placements: [...second] });
    expect(result.current).toBe("running, checks updated");
    expect(startViewTransition).toHaveBeenCalledTimes(1);
    expect(document.documentElement.classList.contains(KANBAN_BOARD_MOTION_ROOT_CLASS)).toBe(true);

    await act(async () => {
      resolveFinished();
      await finished;
    });
    expect(document.documentElement.classList.contains(KANBAN_BOARD_MOTION_ROOT_CLASS)).toBe(false);
  });
});
