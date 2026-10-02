import { describe, expect, it } from "bun:test";
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

  it("uses the integration label without spaces as the docs path", () => {
    expect(
      buildSidebarComponentDocsPayload(
        "listBuckets",
        {
          integrationName: "gcp",
          integrationLabel: "Google Cloud",
        },
        {
          label: "List Buckets",
          payloadLabel: "Example Output",
        },
      ).documentationUrl,
    ).toBe("https://docs.superplane.com/components/googlecloud#list-buckets");
  });

  it("slugifies the integration name when the label is missing", () => {
    expect(
      buildSidebarComponentDocsPayload(
        "getFile",
        { integrationName: "GitHub" },
        {
          label: "Get File",
          payloadLabel: "Example Output",
        },
      ).documentationUrl,
    ).toBe("https://docs.superplane.com/components/git-hub#get-file");
  });

  it("falls back to the block name when no labels are set", () => {
    expect(
      buildSidebarComponentDocsPayload("runCommand", null, {
        payloadLabel: "Example Output",
      }).documentationUrl,
    ).toBe("https://docs.superplane.com/components/core#run-command");
  });

  it("slugifies camelCase, underscores, and dots in the anchor", () => {
    expect(
      buildSidebarComponentDocsPayload("block", null, {
        label: "on_pull.Request",
        payloadLabel: "Example Output",
      }).documentationUrl,
    ).toBe("https://docs.superplane.com/components/core#on-pull-request");
  });

  it("uses unknown when the resolved label is empty", () => {
    expect(
      buildSidebarComponentDocsPayload("   ", null, {
        payloadLabel: "Example Output",
      }).documentationUrl,
    ).toBe("https://docs.superplane.com/components/core#unknown");
  });
});
