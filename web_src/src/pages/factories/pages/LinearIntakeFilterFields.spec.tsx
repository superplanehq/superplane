import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "bun:test";

import { LinearIntakeFilterFields } from "./LinearIntakeFilterFields";
import { DEFAULT_GITHUB_INTAKE_SETTINGS, type IntakeSourceSettings } from "./intakeSourceSettingsModel";
import { LINEAR_INTAKE_SETUP_COPY } from "./linearIntakeSetupCopy";

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: (_organizationId: string, _integrationId: string, resourceType: string) => ({
    data:
      resourceType === "label"
        ? [
            { id: "label-1", name: "bug" },
            { id: "label-2", name: "feature" },
          ]
        : [
            { id: "project-1", name: "Checkout" },
            { id: "project-2", name: "Billing" },
          ],
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));

function Fields({ sourceId = "linear-issues" as const }: { sourceId?: "linear-issues" | "github-issues" }) {
  const [settings, setSettings] = useState<IntakeSourceSettings>({
    ...DEFAULT_GITHUB_INTAKE_SETTINGS,
    linearProjectIds: ["project-1"],
    linearLabels: [],
  });
  return (
    <LinearIntakeFilterFields
      sourceId={sourceId}
      settings={settings}
      onSettingsChange={setSettings}
      organizationId="org-1"
      integrationId="integration-1"
    />
  );
}

describe("LinearIntakeFilterFields", () => {
  it("renders nothing for another source", () => {
    const { container } = render(<Fields sourceId="github-issues" />);
    expect(container).toBeEmptyDOMElement();
  });

  it("selects projects and labels loaded from Linear", async () => {
    const user = userEvent.setup();
    render(<Fields />);

    expect(screen.getByTestId("linear-project-project-1")).toHaveAttribute("aria-selected", "true");
    await user.click(screen.getByTestId("linear-project-project-2"));
    expect(screen.getByTestId("linear-project-project-2")).toHaveAttribute("aria-selected", "true");

    await user.click(screen.getByRole("checkbox", { name: "bug" }));
    expect(screen.getByRole("checkbox", { name: "bug" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "feature" })).not.toBeChecked();
    await user.click(screen.getByRole("checkbox", { name: "bug" }));
    expect(screen.getByRole("checkbox", { name: "bug" })).not.toBeChecked();
    await user.click(screen.getByRole("button", { name: LINEAR_INTAKE_SETUP_COPY.labelNew }));
    await user.type(screen.getByTestId("linear-intake-label-input"), "bug");
    await user.click(screen.getByRole("button", { name: LINEAR_INTAKE_SETUP_COPY.labelAdd }));
    expect(screen.getByText("bug")).toBeInTheDocument();
  });
});
