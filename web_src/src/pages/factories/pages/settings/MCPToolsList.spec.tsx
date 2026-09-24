import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { MCPToolsList } from "./MCPToolsList";

const tools = [
  { name: "search", readOnly: true },
  { name: "create_issue", readOnly: false },
  { name: "write_issue", readOnly: false },
];

describe("MCPToolsList", () => {
  it("shows the enabled count and omits descriptions", () => {
    render(
      <MCPToolsList
        tools={[
          { name: "search", readOnly: true },
          { name: "create_issue", readOnly: false },
          { name: "write_issue", readOnly: false },
        ]}
        isLoading={false}
        isError={false}
        disabledTools={["create_issue"]}
        canUpdate
        onToggleTool={vi.fn()}
      />,
    );

    expect(screen.getByTestId("mcp-tools-list")).toHaveTextContent("2/3");
    expect(screen.getByText("search")).toBeInTheDocument();
    expect(screen.getByText("Read")).toBeInTheDocument();
    expect(screen.getAllByText("Write")).toHaveLength(2);
    expect(screen.queryByText("Search the catalog.")).not.toBeInTheDocument();
  });

  it("sorts write tools first", async () => {
    const user = userEvent.setup();
    render(
      <MCPToolsList
        tools={tools}
        isLoading={false}
        isError={false}
        disabledTools={[]}
        canUpdate
        onToggleTool={vi.fn()}
      />,
    );

    await user.click(screen.getByTestId("mcp-tools-sort"));
    await user.click(screen.getByRole("option", { name: "Write first" }));
    const names = screen.getAllByRole("switch").map((node) => node.getAttribute("aria-label"));
    expect(names).toEqual(["Enable create_issue", "Enable write_issue", "Enable search"]);
  });
});
