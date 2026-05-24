import { describe, expect, it } from "vitest";
import { buildSidebarComponentDocsPayload } from "@/lib/componentDocsUrl";

describe("componentDocsUrl", () => {
  it("builds sidebar payloads with the resolved label and docs url", () => {
    expect(
      buildSidebarComponentDocsPayload(
        "run_workflow",
        {
          displayLabel: "Run Workflow",
          integrationName: "github",
          integrationLabel: "GitHub",
        },
        {
          label: undefined,
          description: "Runs a workflow",
          examplePayload: { ok: true },
          payloadLabel: "Example Output",
        },
      ),
    ).toEqual({
      description: "Runs a workflow",
      examplePayload: { ok: true },
      payloadLabel: "Example Output",
      documentationUrl: "https://docs.superplane.com/components/github#run-workflow",
    });
  });

  it("falls back to the core docs page when there is no integration", () => {
    expect(
      buildSidebarComponentDocsPayload("delay", null, {
        label: "Delay",
        description: undefined,
        examplePayload: undefined,
        payloadLabel: "Example Output",
      }),
    ).toEqual({
      description: undefined,
      examplePayload: undefined,
      payloadLabel: "Example Output",
      documentationUrl: "https://docs.superplane.com/components/core#delay",
    });
  });
});
