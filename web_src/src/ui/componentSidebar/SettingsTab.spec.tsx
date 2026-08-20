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

  describe("original component name metadata", () => {
    it("shows the original component name as secondary metadata only when the node has a genuinely custom name", () => {
      render(
        <SettingsTab
          mode="edit"
          nodeName="Deploy to staging"
          nodeLabel="Send Message"
          blockName="claude.sendMessage"
          configuration={{}}
          configurationFields={[]}
          onSave={vi.fn()}
        />,
      );

      // Custom name remains the primary, editable value
      const nameInput = screen.getByTestId("node-name-input");
      expect(nameInput).toHaveValue("Deploy to staging");

      // Original component display name is shown nearby as secondary metadata
      const originalName = screen.getByTestId("original-component-name");
      expect(originalName.textContent).toContain("Component:");
      expect(originalName.textContent).toContain("Send Message");
      expect(originalName.textContent).not.toContain("Deploy to staging");
    });

    it("does not show the original name when the current name is still the raw/default component key", () => {
      render(
        <SettingsTab
          mode="edit"
          nodeName="github.onPullRequest"
          nodeLabel="On Pull Request"
          blockName="github.onPullRequest"
          configuration={{}}
          configurationFields={[]}
          onSave={vi.fn()}
        />,
      );

      expect(screen.getByTestId("node-name-input")).toHaveValue("github.onPullRequest");
      expect(screen.queryByTestId("original-component-name")).not.toBeInTheDocument();
    });

    it("does not treat the generated unique-name variant of a default component as a rename", () => {
      render(
        <SettingsTab
          mode="edit"
          nodeName="github.onPullRequest 2"
          nodeLabel="On Pull Request"
          blockName="github.onPullRequest"
          configuration={{}}
          configurationFields={[]}
          onSave={vi.fn()}
        />,
      );

      expect(screen.queryByTestId("original-component-name")).not.toBeInTheDocument();
    });

    it("does not show the original name when the custom name is identical to the display label", () => {
      render(
        <SettingsTab
          mode="edit"
          nodeName="Send Message"
          nodeLabel="Send Message"
          blockName="claude.sendMessage"
          configuration={{}}
          configurationFields={[]}
          onSave={vi.fn()}
        />,
      );

      // No redundant/duplicated secondary metadata
      expect(screen.queryByTestId("original-component-name")).not.toBeInTheDocument();
    });

    it("does not crash when the original component metadata is missing or undefined", () => {
      render(
        <SettingsTab
          mode="edit"
          nodeName="Deploy to staging"
          configuration={{}}
          configurationFields={[]}
          onSave={vi.fn()}
        />,
      );

      expect(screen.getByTestId("node-name-input")).toHaveValue("Deploy to staging");
      expect(screen.queryByTestId("original-component-name")).not.toBeInTheDocument();
    });

    it("does not show the original name when no canonical component key is available", () => {
      render(
        <SettingsTab
          mode="edit"
          nodeName="Deploy to staging"
          nodeLabel="Annotation"
          configuration={{}}
          configurationFields={[]}
          onSave={vi.fn()}
        />,
      );

      // Without a canonical component key we cannot prove a rename, so no metadata is shown.
      expect(screen.queryByTestId("original-component-name")).not.toBeInTheDocument();
    });

    it("renders a long original component name without truncating or crashing", () => {
      const longLabel = "Very long original component display name that users might need to read fully";
      render(
        <SettingsTab
          mode="edit"
          nodeName="A"
          nodeLabel={longLabel}
          blockName="github.createIssueComment"
          configuration={{}}
          configurationFields={[]}
          onSave={vi.fn()}
        />,
      );

      expect(screen.getByTestId("original-component-name").textContent).toContain(longLabel);
    });
  });
});
