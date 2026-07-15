import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SettingsTab } from "./SettingsTab";

vi.mock("./configurationView/ConfigurationView", () => ({
  ConfigurationView: () => <div data-testid="configuration-view" />,
}));

describe("SettingsTab", () => {
  it("renders customField in read-only mode", () => {
    render(
      <SettingsTab
        mode="edit"
        nodeName="Wait node"
        configuration={{}}
        configurationFields={[]}
        onSave={vi.fn()}
        readOnly
        customField={() => <div data-testid="custom-field">Wait controls</div>}
      />,
    );

    expect(screen.getByTestId("configuration-view")).toBeInTheDocument();
    expect(screen.getByTestId("custom-field")).toBeInTheDocument();
    expect(screen.queryByTestId("save-node-button")).not.toBeInTheDocument();
  });

  it("renders the disabled form layout when formDisabled even if readOnly", () => {
    render(
      <SettingsTab
        mode="edit"
        nodeName="Wait node"
        configuration={{ namespace: "llmTokenCosts", enabled: true }}
        configurationFields={[
          { name: "namespace", label: "Namespace", type: "string" },
          { name: "enabled", label: "Enabled", type: "boolean" },
        ]}
        onSave={vi.fn()}
        readOnly
        formDisabled
      />,
    );

    expect(screen.queryByTestId("configuration-view")).not.toBeInTheDocument();
    expect(screen.getByTestId("node-name-input")).toBeDisabled();
    expect(screen.getByRole("switch")).toBeDisabled();
    expect(screen.getByTestId("settings-tab-form")).toHaveClass("cursor-not-allowed");
    expect(screen.getByTestId("settings-tab-form")).not.toHaveClass("opacity-70");
  });
});
