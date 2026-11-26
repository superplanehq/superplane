import { describe, expect, it } from "bun:test";

import { getDetailsForAlert } from "./base";
import type { PrometheusAlertPayload } from "./types";

describe("getDetailsForAlert", () => {
  const startsAt = new Date(Date.now() - 60_000).toISOString();
  const endsAt = new Date(Date.now() + 60_000).toISOString();
  const cases: Array<{ name: string; alert: PrometheusAlertPayload; entries: Array<[string, string]> }> = [
    {
      name: "a full alert",
      alert: {
        labels: { alertname: "HighCPU", instance: "server-1", job: "node" },
        status: "resolved",
        annotations: { summary: "CPU usage recovered", description: "CPU usage is below the threshold" },
        startsAt,
        endsAt,
        value: "0",
        generatorURL: "https://prometheus.example.com/graph",
        fingerprint: "alert-1",
        externalURL: "https://alertmanager.example.com",
      },
      entries: [
        ["Alert Name", "HighCPU"],
        ["State", "resolved"],
        ["Instance", "server-1"],
        ["Job", "node"],
        ["Summary", "CPU usage recovered"],
        ["Description", "CPU usage is below the threshold"],
        ["Starts At", new Date(startsAt).toLocaleString()],
        ["Ends At", new Date(endsAt).toLocaleString()],
        ["Value", "0"],
        ["Generator URL", "https://prometheus.example.com/graph"],
        ["Fingerprint", "alert-1"],
        ["Alertmanager URL", "https://alertmanager.example.com"],
      ],
    },
    {
      name: "a sparse alert with missing labels and annotations",
      alert: {},
      entries: [
        ["Alert Name", "-"],
        ["State", "-"],
      ],
    },
    {
      name: "empty optional values and the sentinel end time",
      alert: {
        labels: { alertname: "", instance: "", job: "" },
        status: "",
        annotations: { summary: "", description: "" },
        startsAt: "",
        endsAt: "0001-01-01T00:00:00Z",
        value: "",
        generatorURL: "",
        fingerprint: "",
        externalURL: "",
      },
      entries: [
        ["Alert Name", "-"],
        ["State", "-"],
      ],
    },
  ];

  it.each(cases)("preserves ordered details for $name", ({ alert, entries }) => {
    expect(Object.entries(getDetailsForAlert(alert))).toEqual(entries);
  });
});
