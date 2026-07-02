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
});
