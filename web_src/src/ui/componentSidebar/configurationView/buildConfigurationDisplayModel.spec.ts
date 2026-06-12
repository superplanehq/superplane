import { describe, expect, it } from "vitest";
import {
  settingsTabConfiguration,
  settingsTabFields,
  STORY_INTEGRATION_REF,
  STORY_INTEGRATIONS,
} from "@/ui/configurationFieldRenderer/storybooks/fixtures";
import { buildConfigurationDisplayModel } from "./buildConfigurationDisplayModel";

describe("buildConfigurationDisplayModel", () => {
  it("builds a flat row list with integration and configuration fields", () => {
    const model = buildConfigurationDisplayModel({
      configuration: settingsTabConfiguration,
      configurationFields: settingsTabFields,
      integrationName: "github",
      integrationRef: STORY_INTEGRATION_REF,
      integrations: STORY_INTEGRATIONS,
    });

    expect(model.rows.some((row) => row.key === "nodeName")).toBe(false);
    expect(model.rows.some((row) => row.label === "Instance" && row.displayText === "GitHub Production")).toBe(true);
    expect(model.rows.some((row) => row.integrationStatus === "Ready")).toBe(true);
    expect(model.rows.some((row) => row.label === "Environment" && row.displayText === "Production")).toBe(true);
    expect(model.rows.some((row) => row.label === "Send digest" && row.displayText === "Yes")).toBe(true);
  });

  it("flattens nested object fields with visibility conditions", () => {
    const configurationFields = settingsTabFields.filter((field) => field.name === "authConfig");
    const model = buildConfigurationDisplayModel({
      configuration: {
        authConfig: {
          authMethod: "token",
          token: "sp_live_token",
          includeMetadata: true,
        },
      },
      configurationFields,
    });

    expect(model.rows.some((row) => row.label === "Auth method" && row.displayText === "API token")).toBe(true);
    expect(model.rows.some((row) => row.label === "Token" && row.displayText === "••••••")).toBe(true);
    expect(model.rows.some((row) => row.label === "Username")).toBe(false);
    expect(model.rows.some((row) => row.label === "Include metadata" && row.displayText === "Yes")).toBe(true);
  });

  it("expands list object items", () => {
    const configurationFields = settingsTabFields.filter((field) => field.name === "headers");
    const model = buildConfigurationDisplayModel({
      configuration: {
        headers: [
          { key: "X-Environment", value: "production" },
          { key: "X-Request-Source", value: "storybook" },
        ],
      },
      configurationFields,
    });

    expect(model.rows.some((row) => row.label === "Header 1")).toBe(true);
    expect(model.rows.some((row) => row.label === "Key" && row.displayText === "X-Environment")).toBe(true);
  });

  it("shows not connected integration state", () => {
    const model = buildConfigurationDisplayModel({
      configuration: {},
      configurationFields: [],
      integrationName: "github",
      integrations: [],
    });

    expect(model.rows[0]?.integrationStatus).toBe("Not connected");
  });

  it("does not default to the first integration when no ref is saved", () => {
    const model = buildConfigurationDisplayModel({
      configuration: {},
      configurationFields: [],
      integrationName: "github",
      integrations: STORY_INTEGRATIONS,
    });

    expect(model.rows).toHaveLength(1);
    expect(model.rows[0]?.integrationStatus).toBe("Not connected");
    expect(model.rows.some((row) => row.label === "Instance")).toBe(false);
  });

  it("does not default to the first integration when the saved ref is stale", () => {
    const model = buildConfigurationDisplayModel({
      configuration: {},
      configurationFields: [],
      integrationName: "github",
      integrationRef: { id: "int_deleted", name: "Old GitHub" },
      integrations: STORY_INTEGRATIONS,
    });

    expect(model.rows).toHaveLength(1);
    expect(model.rows[0]?.integrationStatus).toBe("Not connected");
    expect(model.rows.some((row) => row.label === "Instance" && row.displayText === "GitHub Production")).toBe(false);
  });

  it("shows not connected integration type label", () => {
    const model = buildConfigurationDisplayModel({
      configuration: {},
      configurationFields: [],
      integrationName: "github",
      integrations: STORY_INTEGRATIONS,
    });

    expect(model.rows[0]?.displayText).toBe("GitHub");
    expect(model.rows[0]?.integrationStatus).toBe("Not connected");
  });

  it("expands object schema defaults when the object value is missing", () => {
    const configurationFields = settingsTabFields.filter((field) => field.name === "authConfig");
    const model = buildConfigurationDisplayModel({
      configuration: {},
      configurationFields,
    });

    expect(model.rows.some((row) => row.label === "Auth method")).toBe(true);
  });
});
