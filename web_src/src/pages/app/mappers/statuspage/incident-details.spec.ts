import { describe, expect, it } from "bun:test";
import { getDetailsForIncident } from "./utils";
import type { StatuspageIncident } from "./types";
import type { ExecutionInfo } from "../types";

describe("statuspage getDetailsForIncident", () => {
  const now = Date.now();
  const createdAt = new Date(now - 120_000).toISOString();
  const updatedAt = new Date(now - 60_000).toISOString();
  const executionCreatedAt = new Date(now).toISOString();
  const execution: ExecutionInfo = {
    id: "exec-1",
    createdAt: executionCreatedAt,
    updatedAt: executionCreatedAt,
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
  };
  const incident: StatuspageIncident = {
    id: "incident-1",
    name: "API outage",
    status: "investigating",
    impact: "major",
    shortlink: "https://status.example.com/incidents/incident-1",
    created_at: createdAt,
    updated_at: updatedAt,
  };
  const incidentEntries = [
    ["ID", "incident-1"],
    ["Name", "API outage"],
    ["Status", "investigating"],
    ["Impact", "major"],
    ["Incident URL", "https://status.example.com/incidents/incident-1"],
  ];

  it.each([
    ["statuspage.createIncident", "Created At", createdAt],
    ["statuspage.getIncident", "Fetched At", executionCreatedAt],
    ["statuspage.updateIncident", "Updated At", updatedAt],
    [undefined, "Created At", createdAt],
    ["statuspage.unknown", "Started At", executionCreatedAt],
  ] as const)("selects the timestamp for %s before the incident fields", (componentName, key, timestamp) => {
    const details = getDetailsForIncident(incident, {
      componentName,
      execution,
    });

    expect(Object.entries(details)).toEqual([
      [key, new Date(timestamp).toLocaleString()],
      ...incidentEntries,
      ...(componentName === "statuspage.createIncident" ? [] : [["Updates", "No updates recorded"]]),
    ]);
  });

  for (const missingTimestamp of [undefined, ""]) {
    it.each(["statuspage.createIncident", "statuspage.updateIncident", undefined])(
      `uses execution time when incident timestamps are ${String(missingTimestamp)} for %s`,
      (componentName) => {
        const details = getDetailsForIncident(
          { ...incident, created_at: missingTimestamp, updated_at: missingTimestamp },
          { componentName, execution },
        );

        expect(Object.entries(details)).toEqual([
          ["Started At", new Date(executionCreatedAt).toLocaleString()],
          ...incidentEntries,
          ...(componentName === "statuspage.createIncident" ? [] : [["Updates", "No updates recorded"]]),
        ]);
      },
    );

    it.each([
      ["statuspage.createIncident", missingTimestamp, updatedAt],
      ["statuspage.getIncident", createdAt, updatedAt],
      ["statuspage.updateIncident", createdAt, missingTimestamp],
      [undefined, missingTimestamp, missingTimestamp],
      ["statuspage.unknown", createdAt, updatedAt],
    ])("omits timestamps without an eligible source for %s", (componentName, created_at, updated_at) => {
      const details = getDetailsForIncident(
        { ...incident, created_at, updated_at },
        {
          componentName,
          execution: missingTimestamp === undefined ? undefined : { ...execution, createdAt: missingTimestamp },
        },
      );

      expect(Object.entries(details)).toEqual([
        ...incidentEntries,
        ...(componentName === "statuspage.createIncident" ? [] : [["Updates", "No updates recorded"]]),
      ]);
    });
  }

  it("uses creation time when options are absent", () => {
    expect(Object.entries(getDetailsForIncident(incident))).toEqual([
      ["Created At", new Date(createdAt).toLocaleString()],
      ...incidentEntries,
      ["Updates", "No updates recorded"],
    ]);
  });

  const incidentWithUpdates: StatuspageIncident = {
    ...incident,
    incident_updates: [
      { status: "investigating", created_at: new Date(now - 300_000).toISOString(), body: "  Checking requests  " },
      { status: "", body: "  " },
    ],
  };

  it("omits the update timeline for create even when updates exist", () => {
    expect(
      getDetailsForIncident(incidentWithUpdates, { componentName: "statuspage.createIncident" }),
    ).not.toHaveProperty("Updates");
  });

  it.each(["statuspage.getIncident", "statuspage.updateIncident"])(
    "formats the update timeline for %s",
    (componentName) => {
      expect(getDetailsForIncident(incidentWithUpdates, { componentName }).Updates).toBe(
        "investigating (5m ago): Checking requests | - (-): -",
      );
    },
  );

  it("uses the empty-update fallback for an empty update list", () => {
    expect(
      getDetailsForIncident({ ...incident, incident_updates: [] }, { componentName: "statuspage.getIncident" }).Updates,
    ).toBe("No updates recorded");
  });
});
