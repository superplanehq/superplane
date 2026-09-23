import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import { INTAKE_CONNECTION_COPY } from "./intakeConnectionModel";
import { IntakeConnectionFields, type IntakeConnectionFieldsProps } from "./IntakeConnectionFields";

function renderFields(props: Partial<IntakeConnectionFieldsProps> = {}) {
  const onBindingChange = props.onBindingChange ?? vi.fn();
  render(
    <MemoryRouter>
      <IntakeConnectionFields
        sourceId="jira-issues"
        organizationId="org-1"
        integrationsBasePath="/org-1/workspaces/rf/settings/organization/integrations"
        binding={{ integrationId: "", resourceId: "" }}
        integrations={[
          {
            metadata: { id: "jira-1", name: "Atlassian", integrationName: "jira" },
            status: { state: "ready" },
          },
        ]}
        projects={[]}
        onBindingChange={onBindingChange}
        onConnect={vi.fn()}
        {...props}
      />
    </MemoryRouter>,
  );
  return { onBindingChange };
}

describe("IntakeConnectionFields", () => {
  it("shows one integration as a settings link and auto-selects it", async () => {
    const onBindingChange = vi.fn();
    renderFields({ onBindingChange });

    expect(screen.getByText(INTAKE_CONNECTION_COPY.integration)).toBeInTheDocument();
    expect(screen.queryByText(INTAKE_CONNECTION_COPY.chooseJira)).not.toBeInTheDocument();
    const link = screen.getByRole("link", { name: /Atlassian/ });
    expect(link).toHaveAttribute("href", "/org-1/workspaces/rf/settings/organization/integrations/jira-1");
    expect(link.querySelector("img")).not.toBeNull();
    expect(screen.queryByRole("button", { name: /Atlassian/ })).not.toBeInTheDocument();
    await waitFor(() => expect(onBindingChange).toHaveBeenCalledWith({ integrationId: "jira-1", resourceId: "" }));
  });

  it("shows a selectable list when multiple integrations exist", async () => {
    const onBindingChange = vi.fn();
    renderFields({
      organizationId: "org-1",
      onBindingChange,
      integrations: [
        {
          metadata: { id: "jira-1", name: "Atlassian", integrationName: "jira" },
          status: { state: "ready" },
        },
        {
          metadata: { id: "jira-2", name: "Other Jira", integrationName: "jira" },
          status: { state: "ready" },
        },
      ],
    });

    expect(screen.getByText(INTAKE_CONNECTION_COPY.chooseJira)).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /Atlassian/ })).not.toBeInTheDocument();
    expect(screen.getByTestId("intake-connection-jira-1")).toBeInTheDocument();
    expect(screen.getByTestId("intake-connection-jira-2")).toBeInTheDocument();
    expect(onBindingChange).not.toHaveBeenCalled();

    const user = userEvent.setup();
    await user.click(screen.getByTestId("intake-connection-jira-2"));
    expect(onBindingChange).toHaveBeenCalledWith({ integrationId: "jira-2", resourceId: "" });
  });

  it("uses shadcn Select trigger chrome on the project picker", () => {
    renderFields({
      binding: { integrationId: "jira-1", resourceId: "KAN" },
      projects: [{ id: "KAN", name: "setntry-intake-test-project (KAN)" }],
    });

    const trigger = screen.getByTestId("intake-connection-project-select").querySelector("div.relative.flex");
    expect(trigger?.className).toContain("h-8");
    expect(trigger?.className).toContain("bg-white");
    expect(trigger?.className).toContain("border-gray-300");
    expect(trigger?.className).toContain("dark:bg-gray-800");
    expect(trigger?.className).not.toContain("bg-background");
  });
});
