/* eslint-disable @typescript-eslint/no-explicit-any */
import {
  ComponentBaseProps,
  EventSection,
  DEFAULT_EVENT_STATE_MAP,
  EventStateMap,
  EventState,
} from "@/ui/componentBase";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { getTriggerRenderer } from "..";
import terraformIcon from "@/assets/icons/integrations/terraform.svg";
import { MetadataItem } from "@/ui/metadataList";
import {
  ComponentBaseMapper,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
  NodeInfo,
  ExecutionInfo,
  StateFunction,
  EventStateRegistry,
} from "../types";
import { CanvasesCanvasNodeExecution } from "@/api-client";
import { formatTimeAgo } from "@/utils/date";

function extractExpressions(expObj: any, prefixText: string = "\u00A0\u00A0\u00A0\u00A0\u21B3"): any[] {
  const rows: any[] = [];
  if (!expObj || typeof expObj !== "object") return rows;

  const keys = Object.keys(expObj).sort();
  for (const k of keys) {
    let valStr = "";
    const exp = expObj[k];

    if (exp) {
      if (exp.constant_value !== undefined) {
        valStr =
          typeof exp.constant_value === "string" ? `"${exp.constant_value}"` : JSON.stringify(exp.constant_value);
      } else if (exp.references && Array.isArray(exp.references)) {
        valStr = `[${exp.references.join(", ")}]`;
      } else {
        valStr = "<dynamic/computed>";
      }
    }

    if (valStr !== "") {
      rows.push({
        badges: [
          { label: prefixText, bgColor: "bg-transparent", textColor: "text-slate-400" },
          { label: `${k}:`, bgColor: "bg-transparent", textColor: "text-slate-500 font-medium" },
          { label: valStr, bgColor: "bg-transparent", textColor: "text-slate-700 font-mono text-sm break-all" },
        ],
      });
    }
  }
  return rows;
}

export const TERRAFORM_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  running: {
    icon: "loader-circle",
    textColor: "text-gray-800",
    backgroundColor: "bg-blue-100",
    badgeColor: "bg-blue-500",
  },
  needsAttention: {
    icon: "alert-circle",
    textColor: "text-orange-800",
    backgroundColor: "bg-orange-100",
    badgeColor: "bg-orange-500",
  },
  passed: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
};

export const terraformStateFunction: StateFunction = (execution: CanvasesCanvasNodeExecution): EventState => {
  if (!execution) return "neutral";
  if (execution.result === "RESULT_FAILED" || execution.resultReason === "RESULT_REASON_ERROR") return "failed";
  if (execution.result === "RESULT_CANCELLED") return "cancelled";

  const metadata = execution.metadata as Record<string, any>;
  const currentStatus = metadata?.currentStatus;

  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    const needsAttentionStates = [
      "planned",
      "cost_estimated",
      "policy_checked",
      "policy_override",
      "planned_and_saved",
    ];
    if (needsAttentionStates.includes(currentStatus)) {
      return "needsAttention";
    }
    return "running";
  }

  const isFailedState = ["discarded", "errored", "canceled", "policy_soft_failed", "force_canceled"].includes(
    currentStatus,
  );
  if (isFailedState) return "failed";

  return "passed";
};

export const TERRAFORM_STATE_REGISTRY: EventStateRegistry = {
  stateMap: TERRAFORM_STATE_MAP,
  getState: terraformStateFunction,
};

export const terraformComponentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const { nodes, node, componentDefinition, lastExecutions } = context;
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;

    const metadata: MetadataItem[] = [];
    const config = node.configuration as Record<string, any>;
    const nodeMetadata = node.metadata as Record<string, any>;
    if (nodeMetadata?.workspace?.name) {
      metadata.push({ icon: "box", label: nodeMetadata.workspace.name });
    } else if (config?.workspaceId) {
      metadata.push({ icon: "box", label: config.workspaceId });
    }

    const executionMetadata = lastExecution?.metadata as Record<string, any>;
    if (executionMetadata?.runId) {
      metadata.push({ icon: "play", label: executionMetadata.runId });
    }

    // Show plan diff summary if available
    if (
      executionMetadata?.additions !== undefined ||
      executionMetadata?.changes !== undefined ||
      executionMetadata?.destructions !== undefined
    ) {
      const adds = executionMetadata?.additions ?? 0;
      const changes = executionMetadata?.changes ?? 0;
      const destroys = executionMetadata?.destructions ?? 0;
      metadata.push({ icon: "diff", label: `+${adds} ~${changes} -${destroys}` });
    }

    return {
      iconSrc: terraformIcon,
      iconColor: getColorClass(componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      title: node.name || componentDefinition.label || componentDefinition.name || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(nodes, lastExecution) : undefined,
      metadata,
      includeEmptyState: !lastExecution,
      eventStateMap: TERRAFORM_STATE_MAP,
    };
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    const timeStr = timestamp ? formatTimeAgo(new Date(timestamp)) : "";

    const metadata = context.execution.metadata as Record<string, any>;
    if (metadata) {
      const parts: string[] = [];
      if (metadata.workspaceName) parts.push(metadata.workspaceName);
      if (metadata.runId) parts.push(metadata.runId);

      // Add plan diff summary if available
      if (metadata.additions !== undefined || metadata.changes !== undefined || metadata.destructions !== undefined) {
        const adds = metadata.additions ?? 0;
        const changes = metadata.changes ?? 0;
        const destroys = metadata.destructions ?? 0;
        parts.push(`+${adds} ~${changes} -${destroys}`);
      }

      if (parts.length > 0) {
        return timeStr ? `${parts.join(" • ")} • ${timeStr}` : parts.join(" • ");
      }
    }

    return timeStr;
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const details: Record<string, any> = {};
    const metadata = context.execution.metadata as Record<string, any>;

    if (!metadata) return details;

    if (metadata.runId) details["Run ID"] = metadata.runId;
    if (metadata.workspaceName) details["Workspace Name"] = metadata.workspaceName;
    if (metadata.currentStatus) details["Current Status"] = metadata.currentStatus;
    if (metadata.runUrl) details["Run URL"] = metadata.runUrl;

    if (metadata.additions !== undefined) details["Resources Added"] = metadata.additions;
    if (metadata.changes !== undefined) details["Resources Changed"] = metadata.changes;
    if (metadata.destructions !== undefined) details["Resources Destroyed"] = metadata.destructions;

    if (metadata.stateHistory && Array.isArray(metadata.stateHistory) && metadata.stateHistory.length > 0) {
      details["State History"] = {
        __type: "terraformStates",
        states: metadata.stateHistory,
      };
    }

    if (metadata.planJson) {
      try {
        const plan = JSON.parse(metadata.planJson);
        const badgesRows: { badges: { label: string; bgColor: string; textColor: string }[] }[] = [];

        const planDescription = [
          {
            badges: [
              { label: "Terraform Version:", bgColor: "bg-transparent", textColor: "text-slate-500 font-medium" },
              { label: plan.terraform_version || "Unknown", bgColor: "bg-transparent", textColor: "text-slate-800" },
            ],
          },
        ];

        if (plan.timestamp) {
          planDescription.push({
            badges: [
              { label: "Generated At:", bgColor: "bg-transparent", textColor: "text-slate-500 font-medium" },
              {
                label: new Date(plan.timestamp).toLocaleString(),
                bgColor: "bg-transparent",
                textColor: "text-slate-800",
              },
            ],
          });
        }

        let planState = "Complete";
        let stateColor = "text-emerald-800";
        let stateBgColor = "bg-emerald-100";
        if (plan.errored) {
          planState = "Errored";
          stateColor = "text-red-800";
          stateBgColor = "bg-red-100";
        } else if (plan.complete === false) {
          planState = "Incomplete";
          stateColor = "text-amber-800";
          stateBgColor = "bg-amber-100";
        }

        planDescription.push({
          badges: [
            { label: "Plan Status:", bgColor: "bg-transparent", textColor: "text-slate-500 font-medium" },
            { label: planState, bgColor: stateBgColor, textColor: stateColor },
          ],
        });

        planDescription.push({
          badges: [
            { label: "Can Be Applied:", bgColor: "bg-transparent", textColor: "text-slate-500 font-medium" },
            {
              label: plan.applyable ? "Yes" : "No",
              bgColor: plan.applyable ? "bg-emerald-100" : "bg-red-100",
              textColor: plan.applyable ? "text-emerald-800" : "text-red-800",
            },
          ],
        });

        details["Plan Description"] = {
          __type: "expressionBadges",
          values: planDescription,
        };

        if (plan.variables && Object.keys(plan.variables).length > 0) {
          const varRows: any[] = [];
          for (const [key, variable] of Object.entries(plan.variables)) {
            const v = variable as any;
            varRows.push({
              badges: [
                { label: key, bgColor: "bg-slate-100", textColor: "text-slate-800 font-semibold" },
                { label: "=", bgColor: "bg-transparent", textColor: "text-slate-500" },
                {
                  label: JSON.stringify(v.value),
                  bgColor: "bg-transparent",
                  textColor: "text-blue-700 font-mono text-sm break-all",
                },
              ],
            });
          }
          details["Variables"] = {
            __type: "expressionBadges",
            values: varRows,
          };
        }

        if (plan.output_changes && Object.keys(plan.output_changes).length > 0) {
          const outRows: any[] = [];
          for (const [key, outChange] of Object.entries(plan.output_changes)) {
            const o = outChange as any;
            const actions = o.actions || [];
            let actionChar = "~";
            let textColor = "text-amber-600";

            if (actions.includes("create")) {
              actionChar = "+";
              textColor = "text-emerald-600";
            } else if (actions.includes("delete")) {
              actionChar = "-";
              textColor = "text-red-600";
            } else if (actions.includes("replace")) {
              actionChar = "+/-";
              textColor = "text-red-600";
            }

            outRows.push({
              badges: [
                { label: actionChar, bgColor: "bg-transparent", textColor: `${textColor} font-bold text-lg` },
                { label: `output.${key}`, bgColor: "bg-slate-50", textColor: "text-slate-800 font-semibold" },
              ],
            });

            if (actions.includes("create") || actions.includes("update") || actions.includes("replace")) {
              if (o.after_unknown) {
                outRows.push({
                  badges: [
                    { label: `\u00A0\u00A0\u00A0\u00A0${actionChar}`, bgColor: "bg-transparent", textColor },
                    { label: "value :", bgColor: "bg-transparent", textColor: "text-slate-500" },
                    { label: "Known after apply", bgColor: "bg-transparent", textColor: "text-slate-500 italic" },
                  ],
                });
              } else {
                outRows.push({
                  badges: [
                    { label: `\u00A0\u00A0\u00A0\u00A0${actionChar}`, bgColor: "bg-transparent", textColor },
                    { label: "value :", bgColor: "bg-transparent", textColor: "text-slate-500" },
                    {
                      label: typeof o.after === "object" ? "{...}" : JSON.stringify(o.after),
                      bgColor: "bg-transparent",
                      textColor: "text-gray-900 font-semibold",
                    },
                  ],
                });
              }
            } else if (actions.includes("delete")) {
              outRows.push({
                badges: [
                  { label: `\u00A0\u00A0\u00A0\u00A0-`, bgColor: "bg-transparent", textColor },
                  { label: "value :", bgColor: "bg-transparent", textColor: "text-slate-500" },
                  {
                    label: typeof o.before === "object" ? "{...}" : JSON.stringify(o.before),
                    bgColor: "bg-transparent",
                    textColor: "text-gray-900 font-semibold line-through opacity-70",
                  },
                ],
              });
            }
          }
          if (outRows.length > 0) {
            details["Output Changes"] = {
              __type: "expressionBadges",
              values: outRows,
            };
          }
        }

        if (plan.configuration?.provider_config) {
          const providerRows: any[] = [];
          for (const [key, provider] of Object.entries(plan.configuration.provider_config)) {
            const p = provider as any;
            providerRows.push({
              badges: [
                { label: p.name || key, bgColor: "bg-slate-100", textColor: "text-slate-800 font-semibold" },
                { label: p.full_name || "", bgColor: "bg-transparent", textColor: "text-slate-600" },
                {
                  label: p.version_constraint || "latest",
                  bgColor: "bg-blue-50",
                  textColor: "text-blue-700 font-mono text-sm",
                },
              ],
            });

            if (p.expressions && Object.keys(p.expressions).length > 0) {
              providerRows.push(...extractExpressions(p.expressions));
            }
          }
          if (providerRows.length > 0) {
            details["Providers"] = {
              __type: "expressionBadges",
              values: providerRows,
            };
          }
        }

        if (plan.configuration?.root_module?.resources) {
          const modRows: any[] = [];
          for (const res of plan.configuration.root_module.resources) {
            modRows.push({
              badges: [
                { label: "resource", bgColor: "bg-transparent", textColor: "text-slate-500 font-medium" },
                { label: res.type || "unknown", bgColor: "bg-slate-50", textColor: "text-slate-700" },
                { label: res.name || "", bgColor: "bg-transparent", textColor: "text-slate-800 font-semibold" },
              ],
            });

            if (res.expressions && Object.keys(res.expressions).length > 0) {
              modRows.push(...extractExpressions(res.expressions, "\u00A0\u00A0\u00A0\u00A0="));
            }
          }
          if (modRows.length > 0) {
            details["Configured Resources"] = {
              __type: "expressionBadges",
              values: modRows,
            };
          }
        }

        if (plan.resource_changes) {
          for (const res of plan.resource_changes) {
            const actions = res.change?.actions || [];
            if (actions.includes("no-op") || actions.includes("read")) continue;

            let actionChar = "~";
            let textColor = "text-amber-600";
            let rowBgColor = "bg-slate-50";

            if (actions.includes("create")) {
              actionChar = "+";
              textColor = "text-emerald-600";
              rowBgColor = "bg-emerald-50/50";
            } else if (actions.includes("delete")) {
              actionChar = "-";
              textColor = "text-red-600";
              rowBgColor = "bg-red-50/50";
            } else if (actions.includes("replace")) {
              actionChar = "+/-";
              textColor = "text-red-600";
              rowBgColor = "bg-red-50/50";
            }

            badgesRows.push({
              badges: [
                { label: actionChar, bgColor: "bg-transparent", textColor: `${textColor} font-bold text-lg` },
                { label: res.address, bgColor: rowBgColor, textColor: "text-slate-800 font-semibold" },
              ],
            });

            const after = res.change?.after || {};
            const afterUnknown = res.change?.after_unknown || {};
            const before = res.change?.before || {};

            const renderProps = (obj: any, unknownObj: any, prefixChar: string, prefixColor: string) => {
              const keys = new Set([...Object.keys(obj || {}), ...Object.keys(unknownObj || {})]);
              for (const k of Array.from(keys).sort()) {
                let valStr = "";
                let valColor = "text-slate-800";

                if (unknownObj && unknownObj[k] === true) {
                  valStr = "Known after apply";
                  valColor = "text-slate-500 italic";
                } else {
                  const val = obj[k];
                  if (typeof val === "object" && val !== null) {
                    valStr = "{...}";
                    valColor = "text-slate-500";
                  } else if (typeof val === "string") {
                    valStr = `"${val}"`;
                    valColor = "text-gray-900 font-semibold";
                  } else {
                    valStr = String(val);
                    valColor = "text-gray-900 font-semibold";
                  }
                }

                badgesRows.push({
                  badges: [
                    {
                      label: `\u00A0\u00A0\u00A0\u00A0${prefixChar}`,
                      bgColor: "bg-transparent",
                      textColor: prefixColor,
                    },
                    { label: k + " :", bgColor: "bg-transparent", textColor: "text-slate-500" },
                    { label: valStr, bgColor: "bg-transparent", textColor: valColor },
                  ],
                });
              }
            };

            if (actions.includes("create")) {
              renderProps(after, afterUnknown, "+", textColor);
            } else if (actions.includes("delete")) {
              renderProps(before, null, "-", textColor);
            } else if (actions.includes("update") || actions.includes("replace")) {
              renderProps(after, afterUnknown, "~", "text-amber-600");
            }
          }
        }

        if (badgesRows.length > 0) {
          details["Resource Changes"] = {
            __type: "expressionBadges",
            values: badgesRows,
          };
        } else if (plan.resource_changes && plan.resource_changes.length === 0) {
          details["Resource Changes"] = {
            __type: "expressionBadges",
            values: [
              {
                badges: [
                  {
                    label: "No changes. Infrastructure is up-to-date.",
                    bgColor: "bg-transparent",
                    textColor: "text-slate-500 italic",
                  },
                ],
              },
            ],
          };
        }

        return details;
      } catch (e) {
        console.error("Failed to parse Terraform plan.json", e);
      }
    }

    if (metadata.planLog) {
      let logLines = metadata.planLog.split("\n");

      logLines = logLines.filter((l: string) => l.trim() !== "");

      const parsedLines: string[] = [];
      for (const line of logLines) {
        try {
          if (line.trim().startsWith("{") && line.trim().endsWith("}")) {
            const parsed = JSON.parse(line);
            if (parsed["@message"]) {
              parsedLines.push(parsed["@message"]);
            }
          } else {
            parsedLines.push(line);
          }
        } catch {
          parsedLines.push(line);
        }
      }

      if (parsedLines.length > 40) {
        logLines = [
          ...parsedLines.slice(0, 40),
          "...",
          "(Diff is truncated to 40 lines. Click the {} icon above to view the full planLog in the payload)",
        ];
      } else {
        logLines = parsedLines;
      }

      details["Plan Output"] = {
        __type: "expressionBadges",
        values: logLines.map((line: string) => {
          let bgColor = "bg-slate-50";
          let textColor = "text-slate-800";

          if (line.includes("create") || line.includes("add") || line.trim().startsWith("+ ")) {
            bgColor = "bg-emerald-100";
            textColor = "text-emerald-800";
          } else if (
            line.includes("destroy") ||
            line.includes("delete") ||
            line.includes("remove") ||
            line.trim().startsWith("- ")
          ) {
            bgColor = "bg-red-100";
            textColor = "text-red-800";
          } else if (line.includes("update") || line.includes("change") || line.trim().startsWith("~ ")) {
            bgColor = "bg-amber-100";
            textColor = "text-amber-800";
          }

          return {
            badges: [
              {
                label: line.replace(/ /g, "\u00A0") || "\u00A0",
                bgColor,
                textColor,
              },
            ],
          };
        }),
      };
    }

    return details;
  },
};

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName as string);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });
  const executionState = terraformStateFunction(execution);
  const subtitleTimestamp =
    executionState === "running" ? execution.createdAt : execution.updatedAt || execution.createdAt;

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: executionState,
      eventSubtitle: subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : undefined,
      eventId: execution.rootEvent!.id!,
    },
  ];
}
