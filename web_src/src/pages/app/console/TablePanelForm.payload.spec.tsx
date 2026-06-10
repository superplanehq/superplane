import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeAll, describe, it, expect } from "vitest";

import type { SuperplaneComponentsNode } from "@/api-client";

import { ConsoleContextProvider } from "./ConsoleContextProvider";
import type { TablePanelContent } from "./panelTypes";
import { TablePanelForm } from "./TablePanelForm";

const START_NODE: SuperplaneComponentsNode = {
  id: "start-id",
  name: "start",
  type: "TYPE_TRIGGER",
  configuration: {
    templates: [{ name: "deploy", payload: { issue: { number: 0 } } }],
  },
};

const INITIAL: TablePanelContent = {
  title: "",
  dataSource: { kind: "runs", limit: 50 },
  render: {
    kind: "table",
    columns: [{ field: "id", label: "ID" }],
    rowActions: [{ kind: "trigger", label: "Run", node: "start", hook: "run" }],
  },
};

function Harness({ initial, actionPayloadIndex = 0 }: { initial: TablePanelContent; actionPayloadIndex?: number }) {
  const [value, setValue] = useState<TablePanelContent>(initial);
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <ConsoleContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[START_NODE]} canRunNodes>
          <TablePanelForm value={value} onChange={setValue} />
          <pre data-testid="harness-state">
            {JSON.stringify(value.render.rowActions?.[actionPayloadIndex]?.payload ?? null)}
          </pre>
        </ConsoleContextProvider>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

function getActionRemoveButtons(): HTMLButtonElement[] {
  return screen.getAllByRole("button", { name: "Remove row action" });
}

function getPathInputs(): HTMLInputElement[] {
  return screen.getAllByPlaceholderText("data.issue.number") as HTMLInputElement[];
}

function getValueInputs(): HTMLInputElement[] {
  return screen.getAllByPlaceholderText(/int\(value\)/) as HTMLInputElement[];
}

function getPayloadRow(index: number): HTMLElement {
  // Each PayloadEntry renders <div class="grid grid-cols-12 ...">[pathInput][valueInput][removeBtn]</div>.
  // The path input is a bare <input>, so its parent is the row container.
  const path = getPathInputs()[index];
  if (!path) throw new Error(`No payload row at index ${index}`);
  return path.parentElement!;
}

describe("TablePanelForm payload editor", () => {
  beforeAll(() => {
    Element.prototype.scrollIntoView ??= () => {};
  });

  it("renders a trailing blank row by default and adds rows only when typing", () => {
    render(<Harness initial={INITIAL} />);
    expect(getPathInputs()).toHaveLength(1);
    expect(getValueInputs()).toHaveLength(1);
  });

  it("does not duplicate a row when editing the path field (tab/blur friendly)", () => {
    render(<Harness initial={INITIAL} />);
    const [pathInput] = getPathInputs();
    expect(pathInput).toBeTruthy();

    // Type into the path field of the trailing blank row.
    fireEvent.change(pathInput!, { target: { value: "issue.number" } });

    // Should now have a filled row PLUS a new trailing blank — never two filled rows.
    const paths = getPathInputs().map((i) => i.value);
    expect(paths).toEqual(["issue.number", ""]);
  });

  it("commits both halves of a single entry when typing path then value sequentially", () => {
    render(<Harness initial={INITIAL} />);
    const [pathInput] = getPathInputs();
    fireEvent.change(pathInput!, { target: { value: "issue.number" } });

    const valueInput = getValueInputs()[0]!;
    fireEvent.change(valueInput, { target: { value: "{{ pr_number }}" } });

    const state = JSON.parse(screen.getByTestId("harness-state").textContent ?? "null") as Record<
      string,
      string
    > | null;
    expect(state).toEqual({ "issue.number": "{{ pr_number }}" });

    // No duplicate keys: still exactly one filled row + one trailing blank.
    expect(getPathInputs().map((i) => i.value)).toEqual(["issue.number", ""]);
  });

  it("renames a path atomically without leaving the old key behind", () => {
    const seeded: TablePanelContent = {
      ...INITIAL,
      render: {
        ...INITIAL.render,
        rowActions: [
          {
            kind: "trigger",
            label: "Run",
            node: "start",
            hook: "run",
            payload: { foo: "{{ value }}" },
          },
        ],
      },
    };
    render(<Harness initial={seeded} />);
    const [pathInput] = getPathInputs();
    expect(pathInput?.value).toBe("foo");

    fireEvent.change(pathInput!, { target: { value: "bar" } });

    const state = JSON.parse(screen.getByTestId("harness-state").textContent ?? "null") as Record<string, string>;
    expect(state).toEqual({ bar: "{{ value }}" });
    expect(Object.keys(state)).toHaveLength(1);
  });

  it("simulates quick-insert by populating both fields without creating duplicates", () => {
    render(<Harness initial={INITIAL} />);

    const [pathInput] = getPathInputs();
    fireEvent.change(pathInput!, { target: { value: "pr_number" } });
    const valueInput = getValueInputs()[0]!;
    fireEvent.change(valueInput, { target: { value: "{{ pr_number }}" } });

    expect(getPathInputs()).toHaveLength(2);
    expect(getPathInputs()[0]!.value).toBe("pr_number");
    expect(getPathInputs()[1]!.value).toBe("");
  });

  it("removes a row via the trash button", () => {
    const seeded: TablePanelContent = {
      ...INITIAL,
      render: {
        ...INITIAL.render,
        rowActions: [
          {
            kind: "trigger",
            label: "Run",
            node: "start",
            hook: "run",
            payload: { foo: "{{ value }}" },
          },
        ],
      },
    };
    render(<Harness initial={seeded} />);
    expect(getPathInputs()[0]!.value).toBe("foo");

    const filledRow = getPayloadRow(0);
    const removeButton = Array.from(filledRow.querySelectorAll<HTMLButtonElement>("button")).find((b) => !b.disabled)!;
    fireEvent.click(removeButton);

    const state = JSON.parse(screen.getByTestId("harness-state").textContent ?? "null");
    expect(state).toBeNull();
    expect(getPathInputs()).toHaveLength(1);
    expect(getPathInputs()[0]!.value).toBe("");
  });

  it("keeps an in-progress row with an empty value template visible while typing", () => {
    render(<Harness initial={INITIAL} />);
    const [pathInput] = getPathInputs();
    fireEvent.change(pathInput!, { target: { value: "in-progress.path" } });

    // Path filled, template still empty — the row should stay (no auto-deletion).
    expect(getPathInputs().map((i) => i.value)).toEqual(["in-progress.path", ""]);
    const state = JSON.parse(screen.getByTestId("harness-state").textContent ?? "null");
    expect(state).toEqual({ "in-progress.path": "" });
  });

  it("keeps payload drafts aligned when removing a middle row action", () => {
    const threeActions: TablePanelContent = {
      ...INITIAL,
      render: {
        ...INITIAL.render,
        rowActions: [
          { kind: "trigger", label: "Action A", node: "start", hook: "run" },
          { kind: "trigger", label: "Action B", node: "start", hook: "run" },
          { kind: "trigger", label: "Action C", node: "start", hook: "run" },
        ],
      },
    };
    render(<Harness initial={threeActions} actionPayloadIndex={1} />);

    // Each action starts with one trailing blank payload row.
    expect(getPathInputs()).toHaveLength(3);

    // Edit the third action's payload (index 2).
    const thirdActionPath = getPathInputs()[2]!;
    fireEvent.change(thirdActionPath, { target: { value: "keep.after.remove" } });

    expect(getPathInputs().map((i) => i.value)).toEqual(["", "", "keep.after.remove", ""]);

    // Remove the middle action — former index 2 becomes index 1.
    fireEvent.click(getActionRemoveButtons()[1]!);

    expect(screen.getAllByPlaceholderText("Label").map((i) => (i as HTMLInputElement).value)).toEqual([
      "Action A",
      "Action C",
    ]);
    expect(getPathInputs().map((i) => i.value)).toEqual(["", "keep.after.remove", ""]);

    const state = JSON.parse(screen.getByTestId("harness-state").textContent ?? "null");
    expect(state).toEqual({ "keep.after.remove": "" });

    // Adding a new action must not resurrect a stale draft from the old index 2 slot.
    fireEvent.click(screen.getByTestId("table-add-action"));
    expect(screen.getAllByPlaceholderText("Label")).toHaveLength(3);
    const pathValues = getPathInputs().map((i) => i.value);
    expect(pathValues.filter((v) => v === "keep.after.remove")).toHaveLength(1);
    expect(pathValues[pathValues.length - 1]).toBe("");
  });
});
