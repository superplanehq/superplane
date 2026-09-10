import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";

import {
  beginKanbanMotionRoot,
  canStartKanbanViewTransition,
  countKanbanMembershipChanges,
  endKanbanMotionRoot,
  KANBAN_BOARD_MOTION_ROOT_CLASS,
  kanbanBoardSignature,
  kanbanViewTransitionName,
  lineBoardCardPlacements,
  MAX_ANIMATED_KANBAN_CHANGES,
  prefersKanbanReducedMotion,
  shouldAnimateKanbanBoard,
  tasksBoardCardPlacements,
  useKanbanDisplayedBoard,
  type KanbanCardPlacement,
} from "./kanbanCardMotion";
import type { WorkOrderListEntry } from "../lib/workOrderListModel";

const LINE: FactoriesFactoryLine = {
  id: "line-1",
  name: "poc",
  steps: [{ app: { app: "app-plan" } }, { app: { app: "app-build" } }],
};

describe("kanbanViewTransitionName", () => {
  it("prefixes ids so a UUID is a valid CSS custom ident", () => {
    expect(kanbanViewTransitionName("9f2a291e-057d-4159-8eba-b4c59a125df2")).toBe(
      "kanban-wo-9f2a291e-057d-4159-8eba-b4c59a125df2",
    );
  });

  it("replaces characters that are not valid in a custom ident", () => {
    expect(kanbanViewTransitionName("wo/a.b")).toBe("kanban-wo-wo_a_b");
  });
});

describe("kanbanBoardSignature", () => {
  it("stays equal when membership and order stay the same", () => {
    const placements: KanbanCardPlacement[] = [
      { id: "a", column: "backlog" },
      { id: "b", column: "phase-0" },
    ];

    expect(kanbanBoardSignature(placements)).toBe(kanbanBoardSignature([...placements]));
  });

  it("changes when a card moves column or the column order changes", () => {
    const original: KanbanCardPlacement[] = [
      { id: "a", column: "backlog" },
      { id: "b", column: "backlog" },
    ];

    expect(kanbanBoardSignature(original)).not.toBe(
      kanbanBoardSignature([
        { id: "a", column: "phase-0" },
        { id: "b", column: "backlog" },
      ]),
    );
    expect(kanbanBoardSignature(original)).not.toBe(
      kanbanBoardSignature([
        { id: "b", column: "backlog" },
        { id: "a", column: "backlog" },
      ]),
    );
  });
});

describe("countKanbanMembershipChanges", () => {
  it("counts enters, exits, and column moves", () => {
    const previous: KanbanCardPlacement[] = [
      { id: "stay", column: "backlog" },
      { id: "move", column: "backlog" },
      { id: "leave", column: "verify" },
    ];
    const next: KanbanCardPlacement[] = [
      { id: "stay", column: "backlog" },
      { id: "move", column: "phase-0" },
      { id: "enter", column: "backlog" },
    ];

    expect(countKanbanMembershipChanges(previous, next)).toBe(3);
  });

  it("ignores a reorder that keeps the same column", () => {
    expect(
      countKanbanMembershipChanges(
        [
          { id: "a", column: "backlog" },
          { id: "b", column: "backlog" },
        ],
        [
          { id: "b", column: "backlog" },
          { id: "a", column: "backlog" },
        ],
      ),
    ).toBe(0);
  });
});

describe("shouldAnimateKanbanBoard", () => {
  const previous: KanbanCardPlacement[] = [{ id: "a", column: "backlog" }];
  const next: KanbanCardPlacement[] = [{ id: "a", column: "phase-0" }];

  it("skips when the View Transition API is missing", () => {
    expect(
      shouldAnimateKanbanBoard(previous, next, {
        canStartViewTransition: false,
        reducedMotion: false,
      }),
    ).toBe(false);
  });

  it("skips when the user prefers reduced motion", () => {
    expect(
      shouldAnimateKanbanBoard(previous, next, {
        canStartViewTransition: true,
        reducedMotion: true,
      }),
    ).toBe(false);
  });

  it("skips when too many cards change column at once", () => {
    const manyPrevious = Array.from({ length: MAX_ANIMATED_KANBAN_CHANGES + 1 }, (_, index) => ({
      id: `card-${index}`,
      column: "backlog",
    }));
    const manyNext = manyPrevious.map((placement) => ({ ...placement, column: "phase-0" }));

    expect(
      shouldAnimateKanbanBoard(manyPrevious, manyNext, {
        canStartViewTransition: true,
        reducedMotion: false,
      }),
    ).toBe(false);
  });

  it("animates a single column move", () => {
    expect(
      shouldAnimateKanbanBoard(previous, next, {
        canStartViewTransition: true,
        reducedMotion: false,
      }),
    ).toBe(true);
  });
});

describe("prefersKanbanReducedMotion", () => {
  it("reads the reduced-motion media query", () => {
    const matchMedia = vi.fn((query: string) => ({
      matches: query === "(prefers-reduced-motion: reduce)",
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }));

    expect(prefersKanbanReducedMotion(matchMedia)).toBe(true);
  });
});

describe("canStartKanbanViewTransition", () => {
  it("is false when startViewTransition is missing", () => {
    expect(canStartKanbanViewTransition({} as Document)).toBe(false);
  });
});

describe("kanban motion root class", () => {
  function createRoot() {
    const tokens = new Set<string>();
    return {
      classList: {
        add: (token: string) => {
          tokens.add(token);
        },
        remove: (token: string) => {
          tokens.delete(token);
        },
      },
      has: (token: string) => tokens.has(token),
    };
  }

  it("adds the motion class and removes it for a single transition", () => {
    const root = createRoot();
    const generation = beginKanbanMotionRoot(root);

    expect(root.has(KANBAN_BOARD_MOTION_ROOT_CLASS)).toBe(true);
    endKanbanMotionRoot(generation, root);
    expect(root.has(KANBAN_BOARD_MOTION_ROOT_CLASS)).toBe(false);
  });

  it("keeps the class when an older transition ends during a newer one", () => {
    const root = createRoot();
    const first = beginKanbanMotionRoot(root);
    const second = beginKanbanMotionRoot(root);

    endKanbanMotionRoot(first, root);
    expect(root.has(KANBAN_BOARD_MOTION_ROOT_CLASS)).toBe(true);
    endKanbanMotionRoot(second, root);
    expect(root.has(KANBAN_BOARD_MOTION_ROOT_CLASS)).toBe(false);
  });
});

describe("lineBoardCardPlacements", () => {
  it("puts a draft in backlog and an open run on its current phase", () => {
    const workOrders: FactoriesWorkOrder[] = [
      { id: "wo-draft", title: "New", state: "STATE_DRAFT" },
      {
        id: "wo-open",
        title: "In flight",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "dispatch-1",
            line: { id: "line-1", name: "poc" },
            createdAt: "2026-08-11T10:00:00.000Z",
            stepExecutions: [
              {
                id: "e1",
                step: "plan",
                stepIndex: 0,
                state: "STATE_STARTED",
                createdAt: "2026-08-11T10:00:00.000Z",
                updatedAt: "2026-08-11T10:00:00.000Z",
              },
            ],
          },
        ],
      },
    ];

    expect(lineBoardCardPlacements(LINE, workOrders)).toEqual([
      { id: "wo-draft", column: "backlog" },
      { id: "wo-open", column: "phase-0" },
    ]);
  });
});

describe("tasksBoardCardPlacements", () => {
  it("groups entries by the shared status lanes", () => {
    const entries = [
      { id: "done-1", displayStatus: "completed" },
      { id: "backlog-1", displayStatus: "draft" },
      { id: "run-1", displayStatus: "running" },
    ] as WorkOrderListEntry[];

    expect(tasksBoardCardPlacements(entries)).toEqual([
      { id: "backlog-1", column: "backlog" },
      { id: "run-1", column: "running" },
      { id: "done-1", column: "done" },
    ]);
  });
});

describe("useKanbanDisplayedBoard", () => {
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
    const startViewTransition = vi.fn((options: { types?: string[]; update?: () => void }) => {
      queueMicrotask(() => {
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

    try {
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
    } finally {
      vi.unstubAllGlobals();
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
    const startViewTransition = vi.fn((options: { types?: string[]; update?: () => void }) => {
      startCount += 1;
      queueMicrotask(() => {
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

    try {
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
    } finally {
      document.documentElement.classList.remove(KANBAN_BOARD_MOTION_ROOT_CLASS);
      vi.unstubAllGlobals();
    }
  });
});
