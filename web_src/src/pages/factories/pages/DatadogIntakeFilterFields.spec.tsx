import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "bun:test";

import { DatadogIntakeFilterFields } from "./DatadogIntakeFilterFields";
import { GitHubIntakeFilterFields } from "./GitHubIntakeFilterFields";
import { DATADOG_INTAKE_SETTINGS_COPY } from "./datadogIntakeSettingsCopy";
import {
  DEFAULT_DATADOG_INTAKE_SETTINGS,
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  intakeSettingsToApi,
  normalizeIntakeSourceSettings,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: (_organizationId: string, _integrationId: string, resourceType: string) => {
    if (resourceType === "environment") {
      return {
        data: [
          { id: "prod", name: "prod" },
          { id: "staging", name: "staging" },
        ],
        isLoading: false,
        isError: false,
        refetch: vi.fn(),
      };
    }
    return {
      data: [
        { id: "checkout", name: "checkout" },
        { id: "billing", name: "billing" },
      ],
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    };
  },
}));

function FilterHarness({
  sourceId,
  initial,
  onSave,
}: {
  sourceId: LineIntakeSourceId;
  initial: IntakeSourceSettings;
  onSave?: (next: IntakeSourceSettings) => void;
}) {
  const [settings, setSettings] = useState(initial);

  return (
    <div>
      <GitHubIntakeFilterFields sourceId={sourceId} settings={settings} onSettingsChange={setSettings} />
      <DatadogIntakeFilterFields
        sourceId={sourceId}
        settings={settings}
        onSettingsChange={setSettings}
        organizationId="org-1"
        integrationId="datadog-1"
        resourceId={initial.datadogService}
      />
      <button type="button" onClick={() => onSave?.(normalizeIntakeSourceSettings(settings, sourceId))}>
        Save
      </button>
    </div>
  );
}

describe("DatadogIntakeFilterFields", () => {
  it("hides Datadog fields for a GitHub intake", () => {
    render(<FilterHarness sourceId="github-issues" initial={DEFAULT_GITHUB_INTAKE_SETTINGS} />);

    expect(screen.queryByText(DATADOG_INTAKE_SETTINGS_COPY.triggered)).not.toBeInTheDocument();
    expect(screen.queryByTestId("datadog-service-name")).not.toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "A new issue is opened" })).toBeInTheDocument();
  });

  it("shows the service, alert types, and environments for a Datadog intake", () => {
    render(
      <FilterHarness sourceId="datadog" initial={{ ...DEFAULT_DATADOG_INTAKE_SETTINGS, datadogService: "checkout" }} />,
    );

    expect(screen.getByTestId("datadog-service-checkout")).toHaveAttribute("aria-selected", "true");
    expect(screen.queryByTestId("datadog-service-name")).not.toBeInTheDocument();
    expect(screen.queryByText("Set the Error Tracking monitor query to service:<name>.")).not.toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: DATADOG_INTAKE_SETTINGS_COPY.triggered })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: DATADOG_INTAKE_SETTINGS_COPY.retriggered })).not.toBeChecked();
    expect(screen.queryByText("An issue alerts again after it recovered")).not.toBeInTheDocument();
    expect(screen.queryByTestId("datadog-environment-new")).not.toBeInTheDocument();
    expect(screen.getByTestId("datadog-environment-prod")).toBeInTheDocument();
    expect(screen.getByText(DATADOG_INTAKE_SETTINGS_COPY.environmentsHelper)).toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "A new issue is opened" })).not.toBeInTheDocument();
  });

  it("selects an environment from Datadog", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();
    render(
      <FilterHarness
        sourceId="datadog"
        initial={{ ...DEFAULT_DATADOG_INTAKE_SETTINGS, datadogService: "checkout" }}
        onSave={onSave}
      />,
    );

    await user.click(screen.getByTestId("datadog-environment-prod"));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        datadogTriggeredAlerts: true,
        datadogRetriggeredAlerts: false,
        datadogEnvironments: ["prod"],
        datadogService: "checkout",
      }),
    );
    expect(intakeSettingsToApi(onSave.mock.calls[0][0])).toMatchObject({
      datadogTriggeredAlerts: true,
      datadogRetriggeredAlerts: false,
      datadogEnvironments: ["prod"],
    });
  });

  it("saves a Re-Triggered alert selection", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();
    render(
      <FilterHarness
        sourceId="datadog"
        initial={{ ...DEFAULT_DATADOG_INTAKE_SETTINGS, datadogService: "checkout" }}
        onSave={onSave}
      />,
    );

    await user.click(screen.getByRole("checkbox", { name: DATADOG_INTAKE_SETTINGS_COPY.retriggered }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(intakeSettingsToApi(onSave.mock.calls[0][0])).toMatchObject({
      datadogTriggeredAlerts: true,
      datadogRetriggeredAlerts: true,
    });
  });

  it("changes the service from the list", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();
    render(
      <FilterHarness
        sourceId="datadog"
        initial={{ ...DEFAULT_DATADOG_INTAKE_SETTINGS, datadogService: "checkout" }}
        onSave={onSave}
      />,
    );

    await user.click(screen.getByTestId("datadog-service-billing"));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ datadogService: "billing" }));
  });
});
