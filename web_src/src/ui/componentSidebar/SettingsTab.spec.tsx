import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { OrganizationsIntegration } from "@/api-client";
import {
  STORY_DOMAIN_ID,
  STORY_DOMAIN_TYPE,
  STORY_INTEGRATION_REF,
  STORY_INTEGRATIONS,
} from "@/ui/configurationFieldRenderer/storybooks/fixtures";
import { SettingsTab } from "./SettingsTab";

const ERROR_DESCRIPTION =
  "The GitHub App installation needs to be re-authorized before repository metadata can be loaded. Open Configure to finish the browser step and restore access.";

function buildErrorStateIntegrations(): OrganizationsIntegration[] {
  return STORY_INTEGRATIONS.map((integration, index) =>
    index === 0
      ? {
          ...integration,
          status: {
            ...integration.status,
            state: "error",
            stateDescription: ERROR_DESCRIPTION,
          },
        }
      : integration,
  );
}

describe("SettingsTab integration error visibility", () => {
  it("renders error descriptions inline while keeping Configure available and hides the inline error for ready integrations", () => {
    const baseProps = {
      mode: "edit" as const,
      nodeId: "node_renderer_coverage",
      nodeName: "Renderer Coverage Demo",
      configuration: {},
      configurationFields: [],
      onSave: vi.fn(),
      domainId: STORY_DOMAIN_ID,
      domainType: STORY_DOMAIN_TYPE,
      integrationName: "github",
      integrationRef: STORY_INTEGRATION_REF,
      integrationDefinition: {
        name: "github",
        label: "GitHub",
        icon: "github",
      },
      onOpenConfigureIntegrationDialog: vi.fn(),
      configurationSaveMode: "manual" as const,
    };

    const { rerender } = render(<SettingsTab {...baseProps} integrations={buildErrorStateIntegrations()} />);

    expect(screen.getByText(ERROR_DESCRIPTION)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Configure..." })).toBeInTheDocument();

    rerender(<SettingsTab {...baseProps} integrations={STORY_INTEGRATIONS} />);

    expect(screen.queryByText(ERROR_DESCRIPTION)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Configure..." })).toBeInTheDocument();
  });
});
