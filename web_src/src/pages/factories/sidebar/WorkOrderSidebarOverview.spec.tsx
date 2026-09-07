import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { AutomationLink } from "./WorkOrderSidebarOverview";

describe("AutomationLink", () => {
  it("renders an external-link icon next to the automation label when the automation has an app", () => {
    const { getByRole } = render(
      <MemoryRouter>
        <AutomationLink
          organizationId="org-1"
          factoryKey="RF"
          automation={{ nodeId: "node-1", nodeName: "Deploy bot", appId: "app-1" }}
        />
      </MemoryRouter>,
    );

    const link = getByRole("link", { name: "Deploy bot" });
    expect(link).toHaveAttribute("href", "/org-1/workspaces/RF/apps/app-1");
    expect(link.querySelector("svg")).not.toBeNull();
  });

  it("falls back to plain text with no icon when the automation has no app to link to", () => {
    const { queryByRole, getByText } = render(
      <MemoryRouter>
        <AutomationLink
          organizationId="org-1"
          factoryKey="RF"
          automation={{ nodeId: "node-1", nodeName: "Deploy bot" }}
        />
      </MemoryRouter>,
    );

    expect(queryByRole("link")).toBeNull();
    expect(getByText("Deploy bot").querySelector("svg")).toBeNull();
  });
});
