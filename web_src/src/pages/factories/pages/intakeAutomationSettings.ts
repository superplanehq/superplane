import type { CanvasesCanvas, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";

import {
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  normalizeIntakeSourceSettings,
  type IntakeAssignmentFilter,
  type IntakeLabelFilterMode,
  type IntakeListenMode,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

const SETTINGS_METADATA_KEY = "intakeSettings";
const SCORE_EXPRESSION = `int($["Analyze intake"].data[0].result.result)`;

export interface IntakeCanvasSettingsContext {
  sourceId: LineIntakeSourceId;
  triggerNodeId: string;
  analysisNodeId: string;
  createWorkOrderNodeId: string;
}

export function intakeSettingsFromCanvas(
  context: IntakeCanvasSettingsContext,
  canvas: CanvasesCanvas | undefined,
): IntakeSourceSettings {
  const trigger = triggerNode(context, canvas);
  const saved = recordValue(trigger?.metadata?.[SETTINGS_METADATA_KEY]);
  const threshold = thresholdNode(context, canvas);

  return normalizeIntakeSourceSettings({
    ...DEFAULT_GITHUB_INTAKE_SETTINGS,
    name: intakeName(canvas),
    listenMode: listenMode(saved?.listenMode),
    confidencePct: savedConfidence(saved, threshold),
    labelFilterMode: labelFilterMode(saved?.labelFilterMode),
    labels: stringArray(saved?.labels),
    assignment: assignmentFilter(saved?.assignment),
  });
}

function triggerNode(
  context: IntakeCanvasSettingsContext,
  canvas: CanvasesCanvas | undefined,
): ComponentsNode | undefined {
  return canvas?.spec?.nodes?.find((node) => node.id === context.triggerNodeId);
}

function intakeName(canvas: CanvasesCanvas | undefined): string {
  return canvas?.metadata?.name?.trim() || DEFAULT_GITHUB_INTAKE_SETTINGS.name;
}

function savedConfidence(saved: Record<string, unknown> | undefined, threshold: ComponentsNode | undefined): number {
  return numberValue(saved?.confidencePct) ?? confidenceFromExpression(threshold);
}

export function applyIntakeSettingsToCanvas(
  context: IntakeCanvasSettingsContext,
  canvas: CanvasesCanvas,
  settings: IntakeSourceSettings,
): CanvasesCanvas {
  const normalized = normalizeIntakeSourceSettings(settings);
  const threshold = thresholdNode(context, canvas);
  const nodes = (canvas.spec?.nodes ?? []).map((node) => {
    if (node.id === context.triggerNodeId) {
      return withSettingsMetadata(node, normalized);
    }
    if (node.id === threshold?.id) {
      return {
        ...node,
        configuration: {
          ...node.configuration,
          expression: thresholdExpression(context.sourceId, normalized),
        },
      };
    }
    return node;
  });

  return {
    ...canvas,
    metadata: {
      ...canvas.metadata,
      name: normalized.name,
    },
    spec: {
      ...canvas.spec,
      nodes,
      edges: canvas.spec?.edges ?? [],
    },
  };
}

function withSettingsMetadata(node: ComponentsNode, settings: IntakeSourceSettings): ComponentsNode {
  return {
    ...node,
    metadata: {
      ...node.metadata,
      [SETTINGS_METADATA_KEY]: {
        listenMode: settings.listenMode,
        confidencePct: settings.confidencePct,
        labelFilterMode: settings.labelFilterMode,
        labels: settings.labels,
        assignment: settings.assignment,
      },
    },
  };
}

function thresholdNode(
  context: IntakeCanvasSettingsContext,
  canvas: CanvasesCanvas | undefined,
): ComponentsNode | undefined {
  const edges = canvas?.spec?.edges ?? [];
  const nodes = canvas?.spec?.nodes ?? [];
  const afterAnalysis = new Set(
    edges
      .filter((edge) => edge.sourceId === context.analysisNodeId)
      .flatMap((edge) => (edge.targetId ? [edge.targetId] : [])),
  );
  return nodes.find((node) => node.component === "if" && afterAnalysis.has(node.id ?? ""));
}

function thresholdExpression(sourceId: LineIntakeSourceId, settings: IntakeSourceSettings): string {
  const conditions = [`${SCORE_EXPRESSION} >= ${settings.confidencePct}`];
  if (sourceId !== "github-issues") {
    return conditions[0];
  }

  if (settings.labels.length > 0) {
    const labels = JSON.stringify(settings.labels);
    const matchesLabels = `root().data.issue.labels.exists(label, label.name in ${labels})`;
    conditions.push(settings.labelFilterMode === "include" ? matchesLabels : `!(${matchesLabels})`);
  }
  if (settings.assignment === "assigned") {
    conditions.push("size(root().data.issue.assignees) > 0");
  }
  if (settings.assignment === "unassigned") {
    conditions.push("size(root().data.issue.assignees) == 0");
  }
  return conditions.join(" && ");
}

function confidenceFromExpression(node: ComponentsNode | undefined): number {
  const expression = typeof node?.configuration?.expression === "string" ? node.configuration.expression : "";
  const match = expression.match(/>=\s*(\d{1,3})/);
  return match ? Number(match[1]) : DEFAULT_GITHUB_INTAKE_SETTINGS.confidencePct;
}

function recordValue(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === "object" && !Array.isArray(value) ? (value as Record<string, unknown>) : undefined;
}

function numberValue(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function stringArray(value: unknown): string[] {
  return Array.isArray(value)
    ? value.flatMap((item) => (typeof item === "string" && item.trim() ? [item.trim()] : []))
    : [];
}

function listenMode(value: unknown): IntakeListenMode {
  return value === "schedule" ? "schedule" : "listen";
}

function labelFilterMode(value: unknown): IntakeLabelFilterMode {
  return value === "exclude" ? "exclude" : "include";
}

function assignmentFilter(value: unknown): IntakeAssignmentFilter {
  return value === "assigned" || value === "unassigned" ? value : "any";
}
