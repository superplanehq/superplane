import { describe, expect, it } from "vitest";

import { sortConnectedIntegrationsByType } from "@/lib/sortConnectedIntegrations";

describe("sortConnectedIntegrationsByType", () => {
  it("groups integrations of the same type next to each other", () => {
    const integrations = [
      { metadata: { id: "int-slack-2", name: "slack-eng", integrationName: "slack" } },
      { metadata: { id: "int-circleci-2", name: "circleci-staging", integrationName: "circleci" } },
      { metadata: { id: "int-slack-1", name: "slack-alerts", integrationName: "slack" } },
      { metadata: { id: "int-circleci-1", name: "circleci-prod", integrationName: "circleci" } },
    ];

    const sorted = sortConnectedIntegrationsByType(integrations);

    expect(sorted.map((integration) => integration.metadata.id)).toEqual([
      "int-circleci-1",
      "int-circleci-2",
      "int-slack-1",
      "int-slack-2",
    ]);
  });

  it("sorts case-insensitively by type", () => {
    const integrations = [
      { metadata: { id: "int-2", name: "b", integrationName: "Slack" } },
      { metadata: { id: "int-1", name: "a", integrationName: "circleci" } },
    ];

    const sorted = sortConnectedIntegrationsByType(integrations);

    expect(sorted.map((integration) => integration.metadata.id)).toEqual(["int-1", "int-2"]);
  });

  it("falls back to id when type and name are equal, for full determinism", () => {
    const integrations = [
      { metadata: { id: "int-b", name: "same-name", integrationName: "circleci" } },
      { metadata: { id: "int-a", name: "same-name", integrationName: "circleci" } },
    ];

    const sorted = sortConnectedIntegrationsByType(integrations);

    expect(sorted.map((integration) => integration.metadata.id)).toEqual(["int-a", "int-b"]);
  });

  it("does not mutate the input array", () => {
    const integrations = [
      { metadata: { id: "int-b", name: "b", integrationName: "slack" } },
      { metadata: { id: "int-a", name: "a", integrationName: "circleci" } },
    ];
    const original = [...integrations];

    sortConnectedIntegrationsByType(integrations);

    expect(integrations).toEqual(original);
  });
});
