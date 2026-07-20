import { describe, expect, it } from "vitest";

import type { CanvasesCanvas } from "@/api-client";

import { buildAppFiles } from "../files/lib/app-files";

import {
  buildCanvasYamlExportPayload,
  dematerializeCanvasMetadata,
  dematerializeCanvasSpec,
  materializeCanvasSpec,
  parseCanvasYamlForImport,
} from "./workflow-spec-files";
import { CANVAS_YAML_PATH } from "./workflow-spec-paths";

const sampleWorkflow: CanvasesCanvas = {
  metadata: {
    id: "canvas-abc",
    name: "My Workflow",
    description: "",
  },
  spec: {
    nodes: [
      {
        id: "node-1",
        name: "Start",
        type: "TYPE_TRIGGER",
        component: "schedule",
        position: { x: 0, y: 0 },
      },
    ],
    edges: [],
  },
};

describe("workflow-spec-files", () => {
  it("materialize and dematerialize canvas spec round-trip", () => {
    const yamlText = materializeCanvasSpec(sampleWorkflow);
    const spec = dematerializeCanvasSpec(yamlText);

    expect(spec?.nodes).toHaveLength(1);
    expect(spec?.nodes?.[0]?.id).toBe("node-1");
  });

  it("Files tab canvas.yaml matches export payload", () => {
    const exportPayload = buildCanvasYamlExportPayload(sampleWorkflow);
    const files = buildAppFiles({
      canvas: sampleWorkflow,
      panels: [],
      layout: [],
      canvasId: sampleWorkflow.metadata?.id,
      canvasName: sampleWorkflow.metadata?.name,
      consoleLoading: false,
      consoleError: null,
    });

    const canvasFile = files.find((file) => file.path === CANVAS_YAML_PATH);
    expect(canvasFile?.content).toBe(exportPayload?.yamlText);
  });

  it("uses canvas name for export download filename", () => {
    expect(buildCanvasYamlExportPayload(sampleWorkflow)?.filename).toBe("my-workflow.yaml");
  });

  it("parses canvas yaml for import with validation errors", () => {
    expect(parseCanvasYamlForImport("")).toEqual({ ok: false, error: "Please provide a YAML definition." });
    expect(parseCanvasYamlForImport("kind: Workflow\nspec:\n  nodes: []").ok).toBe(false);
  });

  it("dematerializes canvas-level metadata from canvas.yaml", () => {
    const yamlText = materializeCanvasSpec(sampleWorkflow);
    expect(dematerializeCanvasMetadata(yamlText)).toEqual({
      id: "canvas-abc",
      name: "My Workflow",
      description: "",
    });
  });

  it("returns null canvas metadata for empty or non-canvas yaml", () => {
    expect(dematerializeCanvasMetadata("")).toBeNull();
    expect(dematerializeCanvasMetadata("kind: Workflow\nmetadata:\n  name: Nope")).toBeNull();
  });
});
