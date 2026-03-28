import type { ComponentsNode } from "@/api-client";

export type LintSeverity = "error" | "warning" | "info";
export type QualityGrade = "A" | "B" | "C" | "D" | "F";

export interface LintIssue {
  severity: LintSeverity;
  rule: string;
  nodeId: string;
  nodeName: string;
  message: string;
}

export interface LintResult {
  status: "pass" | "fail";
  errors: LintIssue[];
  warnings: LintIssue[];
  info: LintIssue[];
  errorCount: number;
  warningCount: number;
  infoCount: number;
  qualityScore: number;
  qualityGrade: QualityGrade;
}

/** Accepts either ComponentsEdge (from API spec) or React Flow Edge shape. */
export interface LintEdge {
  sourceId?: string;
  targetId?: string;
  source?: string;
  target?: string;
  channel?: string;
}

function edgeSourceId(e: LintEdge): string | undefined {
  return e.sourceId || e.source;
}
function edgeTargetId(e: LintEdge): string | undefined {
  return e.targetId || e.target;
}

const TERMINAL_COMPONENTS = new Set([
  "approval",
  "slack.sendTextMessage",
  "slack.waitForButtonClick",
  "github.createIssue",
  "github.createIssueComment",
  "github.createRelease",
  "github.updateIssue",
  "github.publishCommitStatus",
  "github.addReaction",
  "pagerduty.createIncident",
  "pagerduty.resolveIncident",
  "pagerduty.escalateIncident",
  "pagerduty.annotateIncident",
  "pagerduty.acknowledgeIncident",
]);

const DESTRUCTIVE_COMPONENTS = new Set([
  "pagerduty.resolveIncident",
  "pagerduty.escalateIncident",
  "github.deleteRelease",
  "github.createRelease",
]);

const NODE_REF_DOUBLE = /\$\["([^"]+)"\]/g;
const NODE_REF_SINGLE = /\$\['([^']+)'\]/g;

function getComponentName(node: ComponentsNode): string {
  return node.component?.name || node.trigger?.name || "";
}

function computeQualityScore(
  errors: number,
  warnings: number,
  infos: number,
): { score: number; grade: QualityGrade } {
  const ep = Math.min(errors * 15, 60);
  const wp = Math.min(warnings * 5, 30);
  const ip = Math.min(infos * 1, 10);
  const score = Math.max(0, 100 - ep - wp - ip);

  let grade: QualityGrade;
  if (score >= 90) grade = "A";
  else if (score >= 75) grade = "B";
  else if (score >= 60) grade = "C";
  else if (score >= 40) grade = "D";
  else grade = "F";

  return { score, grade };
}

/** Recursively collect all string values from a config object. */
function collectStrings(obj: unknown): string[] {
  if (typeof obj === "string") return [obj];
  if (Array.isArray(obj)) return obj.flatMap(collectStrings);
  if (obj && typeof obj === "object") {
    return Object.values(obj).flatMap(collectStrings);
  }
  return [];
}

export function lintCanvas(
  nodes: ComponentsNode[] | undefined,
  edges: LintEdge[] | undefined,
): LintResult {
  const result: LintResult = {
    status: "pass",
    errors: [],
    warnings: [],
    info: [],
    errorCount: 0,
    warningCount: 0,
    infoCount: 0,
    qualityScore: 100,
    qualityGrade: "A",
  };

  if (!nodes?.length) return result;

  const safeEdges = edges || [];
  const nodeById = new Map(nodes.map((n) => [n.id, n]));
  const nodeNames = new Set(nodes.map((n) => n.name));
  const widgets = new Set(nodes.filter((n) => n.type === "TYPE_WIDGET").map((n) => n.id));
  const triggers = nodes.filter((n) => n.type === "TYPE_TRIGGER");

  // Build adjacency.
  const outgoing = new Map<string, LintEdge[]>();
  const incoming = new Map<string, LintEdge[]>();
  for (const e of safeEdges) {
    const src = edgeSourceId(e);
    const tgt = edgeTargetId(e);
    if (src) {
      const list = outgoing.get(src) || [];
      list.push(e);
      outgoing.set(src, list);
    }
    if (tgt) {
      const list = incoming.get(tgt) || [];
      list.push(e);
      incoming.set(tgt, list);
    }
  }

  // ---- Rule: Duplicate node IDs ----
  const seenIds = new Set<string>();
  for (const n of nodes) {
    if (n.id && seenIds.has(n.id)) {
      result.errors.push({
        severity: "error",
        rule: "duplicate-node-id",
        nodeId: n.id,
        nodeName: n.name || "",
        message: `Duplicate node ID "${n.id}"`,
      });
    }
    if (n.id) seenIds.add(n.id);
  }

  // ---- Rule: Duplicate node names (non-widgets) ----
  const seenNames = new Set<string>();
  for (const n of nodes) {
    if (widgets.has(n.id!)) continue;
    if (n.name && seenNames.has(n.name)) {
      result.warnings.push({
        severity: "warning",
        rule: "duplicate-node-name",
        nodeId: n.id || "",
        nodeName: n.name,
        message: `Duplicate node name "${n.name}" — expression references may be ambiguous`,
      });
    }
    if (n.name) seenNames.add(n.name);
  }

  // ---- Rule: Invalid edges ----
  const seenEdgeKeys = new Set<string>();
  for (let i = 0; i < safeEdges.length; i++) {
    const e = safeEdges[i];
    const src = edgeSourceId(e);
    const tgt = edgeTargetId(e);

    if (src && !nodeById.has(src)) {
      result.errors.push({
        severity: "error",
        rule: "invalid-edge",
        nodeId: src,
        nodeName: "",
        message: `Edge ${i} references nonexistent source node "${src}"`,
      });
    }
    if (tgt && !nodeById.has(tgt)) {
      result.errors.push({
        severity: "error",
        rule: "invalid-edge",
        nodeId: tgt || "",
        nodeName: "",
        message: `Edge ${i} references nonexistent target node "${tgt}"`,
      });
    }
    if (src && tgt && src === tgt) {
      result.errors.push({
        severity: "error",
        rule: "invalid-edge",
        nodeId: src,
        nodeName: nodeById.get(src)?.name || "",
        message: `Edge ${i} is a self-loop on node "${src}"`,
      });
    }
    if (src && tgt) {
      const key = `${src}|${tgt}|${e.channel || "default"}`;
      if (seenEdgeKeys.has(key)) {
        result.warnings.push({
          severity: "warning",
          rule: "duplicate-edge",
          nodeId: src,
          nodeName: nodeById.get(src)?.name || "",
          message: `Duplicate edge from "${src}" to "${tgt}" on channel "${e.channel || "default"}"`,
        });
      }
      seenEdgeKeys.add(key);
    }
    if (src && widgets.has(src)) {
      result.errors.push({
        severity: "error",
        rule: "invalid-edge",
        nodeId: src,
        nodeName: nodeById.get(src)?.name || "",
        message: `Edge ${i} uses widget node "${src}" as source`,
      });
    }
    if (tgt && widgets.has(tgt)) {
      result.errors.push({
        severity: "error",
        rule: "invalid-edge",
        nodeId: tgt,
        nodeName: nodeById.get(tgt)?.name || "",
        message: `Edge ${i} uses widget node "${tgt}" as target`,
      });
    }
  }

  // ---- Rule: Cycle detection (Kahn's) ----
  const inDegree = new Map<string, number>();
  const adj = new Map<string, string[]>();
  for (const n of nodes) {
    if (widgets.has(n.id!)) continue;
    inDegree.set(n.id!, 0);
  }
  for (const e of safeEdges) {
    const src = edgeSourceId(e);
    const tgt = edgeTargetId(e);
    if (!src || !tgt) continue;
    if (widgets.has(src) || widgets.has(tgt)) continue;
    adj.set(src, [...(adj.get(src) || []), tgt]);
    inDegree.set(tgt, (inDegree.get(tgt) || 0) + 1);
  }
  const kahnQueue: string[] = [];
  for (const [id, deg] of inDegree) {
    if (deg === 0) kahnQueue.push(id);
  }
  let kahnVisited = 0;
  while (kahnQueue.length > 0) {
    const cur = kahnQueue.shift()!;
    kahnVisited++;
    for (const next of adj.get(cur) || []) {
      const d = (inDegree.get(next) || 1) - 1;
      inDegree.set(next, d);
      if (d === 0) kahnQueue.push(next);
    }
  }
  const totalNonWidget = nodes.filter((n) => !widgets.has(n.id!)).length;
  if (kahnVisited < totalNonWidget) {
    result.errors.push({
      severity: "error",
      rule: "cycle-detected",
      nodeId: "",
      nodeName: "",
      message: "Cycle detected in canvas graph",
    });
  }

  // ---- Rule: Orphan nodes ----
  const reachable = new Set<string>();
  const bfsQueue = triggers.map((t) => t.id!).filter(Boolean);
  for (const id of bfsQueue) reachable.add(id);
  while (bfsQueue.length > 0) {
    const current = bfsQueue.shift()!;
    for (const e of outgoing.get(current) || []) {
      const tgt = edgeTargetId(e);
      if (tgt && !reachable.has(tgt)) {
        reachable.add(tgt);
        bfsQueue.push(tgt);
      }
    }
  }
  for (const n of nodes) {
    if (widgets.has(n.id!) || reachable.has(n.id!)) continue;
    result.warnings.push({
      severity: "warning",
      rule: "orphan-node",
      nodeId: n.id || "",
      nodeName: n.name || "",
      message: `Node "${n.name}" is not reachable from any trigger`,
    });
  }

  // ---- Rule: Dead ends ----
  for (const n of nodes) {
    if (widgets.has(n.id!) || n.type === "TYPE_TRIGGER") continue;
    if ((outgoing.get(n.id!) || []).length > 0) continue;
    if (TERMINAL_COMPONENTS.has(getComponentName(n))) continue;
    result.warnings.push({
      severity: "warning",
      rule: "dead-end",
      nodeId: n.id || "",
      nodeName: n.name || "",
      message: `Node "${n.name}" has no outgoing edges and is not a terminal component`,
    });
  }

  // ---- Rule: Missing approval gate ----
  for (const n of nodes) {
    if (widgets.has(n.id!)) continue;
    const comp = getComponentName(n);
    if (!DESTRUCTIVE_COMPONENTS.has(comp)) continue;

    const visited = new Set<string>([n.id!]);
    const rQueue = [n.id!];
    let found = false;
    while (rQueue.length > 0 && !found) {
      const cur = rQueue.shift()!;
      for (const e of incoming.get(cur) || []) {
        const srcId = edgeSourceId(e);
        if (!srcId || visited.has(srcId)) continue;
        visited.add(srcId);
        const src = nodeById.get(srcId);
        if (src && getComponentName(src) === "approval") {
          found = true;
          break;
        }
        rQueue.push(srcId);
      }
    }
    if (!found) {
      result.errors.push({
        severity: "error",
        rule: "missing-approval-gate",
        nodeId: n.id || "",
        nodeName: n.name || "",
        message: `Destructive action "${comp}" in "${n.name}" has no upstream approval gate`,
      });
    }
  }

  // ---- Rule: Missing required config ----
  for (const n of nodes) {
    const comp = getComponentName(n);
    const config = (n.configuration || {}) as Record<string, unknown>;

    switch (comp) {
      case "claude.textPrompt": {
        const prompt = typeof config.prompt === "string" ? config.prompt.trim() : "";
        if (!prompt) {
          result.errors.push({
            severity: "error",
            rule: "missing-required-config",
            nodeId: n.id || "",
            nodeName: n.name || "",
            message: `Node "${n.name}" (claude.textPrompt) is missing required "prompt" configuration`,
          });
        }
        break;
      }
      case "slack.sendTextMessage": {
        if (!config.channel) {
          result.warnings.push({
            severity: "warning",
            rule: "missing-required-config",
            nodeId: n.id || "",
            nodeName: n.name || "",
            message: `Node "${n.name}" (slack.sendTextMessage) is missing "channel" configuration`,
          });
        }
        break;
      }
      case "merge": {
        const inCount = (incoming.get(n.id!) || []).length;
        if (inCount < 2) {
          result.info.push({
            severity: "info",
            rule: "missing-required-config",
            nodeId: n.id || "",
            nodeName: n.name || "",
            message: `Node "${n.name}" (merge) has ${inCount} incoming edge(s); merge typically expects 2 or more`,
          });
        }
        break;
      }
      case "filter": {
        const expr = typeof config.expression === "string" ? config.expression.trim() : "";
        if (!expr) {
          result.errors.push({
            severity: "error",
            rule: "missing-required-config",
            nodeId: n.id || "",
            nodeName: n.name || "",
            message: `Node "${n.name}" (filter) is missing required "expression" configuration`,
          });
        }
        break;
      }
      case "http": {
        if (!config.url) {
          result.warnings.push({
            severity: "warning",
            rule: "missing-required-config",
            nodeId: n.id || "",
            nodeName: n.name || "",
            message: `Node "${n.name}" (http) is missing "url" configuration`,
          });
        }
        break;
      }
    }
  }

  // ---- Rule: Expression syntax validation ----
  for (const n of nodes) {
    if (widgets.has(n.id!) || !n.configuration) continue;
    const strings = collectStrings(n.configuration);
    for (const val of strings) {
      const openCount = (val.match(/\{\{/g) || []).length;
      const closeCount = (val.match(/\}\}/g) || []).length;
      if (openCount !== closeCount) {
        result.errors.push({
          severity: "error",
          rule: "invalid-expression",
          nodeId: n.id || "",
          nodeName: n.name || "",
          message: `Node "${n.name}" has unbalanced expression delimiters: ${openCount} '{{' vs ${closeCount} '}}'`,
        });
      }

      for (const pat of [NODE_REF_DOUBLE, NODE_REF_SINGLE]) {
        pat.lastIndex = 0;
        let m;
        while ((m = pat.exec(val)) !== null) {
          if (!nodeNames.has(m[1])) {
            result.warnings.push({
              severity: "warning",
              rule: "invalid-expression",
              nodeId: n.id || "",
              nodeName: n.name || "",
              message: `Node "${n.name}" references unknown node "${m[1]}"`,
            });
          }
        }
      }
    }
  }

  // ---- Rule: Unreachable branches ----
  for (const n of nodes) {
    if (getComponentName(n) !== "filter") continue;
    const edges = outgoing.get(n.id!) || [];
    const hasDefault = edges.some((e) => e.channel === "default");
    if (!hasDefault) {
      result.info.push({
        severity: "info",
        rule: "unreachable-branch",
        nodeId: n.id || "",
        nodeName: n.name || "",
        message: `Filter node "${n.name}" has no "default" channel outgoing edge; matched events have nowhere to go`,
      });
    }
  }

  // Compute counts and quality score.
  result.errorCount = result.errors.length;
  result.warningCount = result.warnings.length;
  result.infoCount = result.info.length;
  result.status = result.errorCount > 0 ? "fail" : "pass";

  const qs = computeQualityScore(result.errorCount, result.warningCount, result.infoCount);
  result.qualityScore = qs.score;
  result.qualityGrade = qs.grade;

  return result;
}
