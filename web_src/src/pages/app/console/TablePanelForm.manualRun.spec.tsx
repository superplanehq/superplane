import { fireEvent, render, screen, within } from "@testing-library/react";
import { useState } from "react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeAll, describe, expect, it } from "vitest";

import type { SuperplaneComponentsNode } from "@/api-client";

import { ConsoleContextProvider } from "./ConsoleContextProvider";
import type { TablePanelContent } from "./panelTypes";
import { TablePanelForm } from "./TablePanelForm";

const START_NODE: SuperplaneComponentsNode = {
  id: "start-id",
  name: "start",
  type: "TYPE_TRIGGER",
  component: "start",
};

const PR_NODE: SuperplaneComponentsNode = {
  id: "pr-id",
  name: "on-pr",
  type: "TYPE_TRIGGER",
  component: "github.onPullRequest",
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

function Harness() {
  const [value, setValue] = useState<TablePanelContent>(INITIAL);
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <ConsoleContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[START_NODE, PR_NODE]} canRunNodes>
          <TablePanelForm value={value} onChange={setValue} />
        </ConsoleContextProvider>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("TablePanelForm trigger node dropdown", () => {
  beforeAll(() => {
    Element.prototype.scrollIntoView ??= () => {};
    // Radix Select uses pointer capture APIs that jsdom does not implement.
    Element.prototype.hasPointerCapture ??= () => false;
    Element.prototype.setPointerCapture ??= () => {};
    Element.prototype.releasePointerCapture ??= () => {};
  });

  function openTriggerNodeSelect() {
    // The action row's node select is the second Select trigger (after the
    // memory namespace picker). Filter down to the one whose current value
    // matches the row's node reference.
    const combos = screen.getAllByRole("combobox");
    const nodeSelect = combos.find((el) => within(el).queryByText("start"));
    if (!nodeSelect) throw new Error("Trigger node select not found");
    fireEvent.pointerDown(nodeSelect, { button: 0, ctrlKey: false });
    fireEvent.click(nodeSelect);
    return nodeSelect;
  }

  it("only lists manually runnable triggers (start / schedule)", () => {
    render(<Harness />);
    openTriggerNodeSelect();
    const listbox = screen.getByRole("listbox");
    expect(within(listbox).getByText("start")).toBeInTheDocument();
    expect(within(listbox).queryByText("on-pr")).toBeNull();
  });
});
