/* eslint-disable @typescript-eslint/no-explicit-any */
import { resolveIcon, isUrl, calcRelativeTimeFromDiff } from "@/lib/utils";
import React, { useCallback, useMemo, useState } from "react";
import {
  DEFAULT_EVENT_STATE_MAP,
  EventState,
  EventStateMap,
  EventStateStyle,
  ComponentBaseSpecValue,
} from "@/ui/componentBase";
import { CanvasesCanvasNodeExecution, ComponentsNode, CanvasesCanvasEvent } from "@/api-client";
import JsonView from "@uiw/react-json-view";
import { SimpleTooltip } from "../componentSidebar/SimpleTooltip";
import { formatTimeAgo } from "@/utils/date";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { getComponentBaseMapper } from "@/pages/workflowv2/mappers";
import { buildExecutionInfo, buildNodeInfo } from "@/pages/workflowv2/utils";

export interface ChildExecution {
  name: string;
  state: string;
  nodeId: string;
  executionId: string;
  badgeColor?: string;
  backgroundColor?: string;
  componentIcon?: string;
}

export interface ChainItemData {
  id: string;
  nodeId: string;
  componentName: string;
  nodeName?: string;
  nodeDisplayName?: string; // The actual display name from workflow node
  nodeIcon?: string;
  nodeIconSlug?: string; // Icon slug from component/trigger/blueprint metadata
  state?: string; // Make state optional since it will be calculated
  executionId?: string;
  originalExecution?: CanvasesCanvasNodeExecution; // Add execution data
  originalEvent?: CanvasesCanvasEvent; // Add event data for trigger events
  childExecutions?: ChildExecution[]; // Add child executions for composite components
  workflowNode?: ComponentsNode; // Add workflow node for subtitle generation
  additionalData?: unknown; // Add additional data for subtitle generation
  tabData?: {
    current?: Record<string, any>;
    payload?: any;
  };
}

type DetailValue = {
  text: string;
  comment?: string;
};

type ExpressionBadges = {
  __type: "expressionBadges";
  values: ComponentBaseSpecValue[];
};

type EvaluationBadges = {
  __type: "evaluationBadges";
  values: ComponentBaseSpecValue[];
  passed: boolean;
  failedParts?: string[];
};

type ErrorValue = {
  __type: "error";
  message: string;
};

type ApprovalTimelineEntry = {
  label: string;
  status: string;
  timestamp?: string;
  comment?: string;
};

type IssueListEntry = {
  status: "degraded" | "critical";
  checkName: string;
  checkSummary?: string;
  checkDescription?: string;
};

type SemaphoreJobEntry = {
  name?: string;
  result?: string;
  status?: string;
};

type SemaphoreBlockEntry = {
  name?: string;
  result?: string;
  resultReason?: string;
  state?: string;
  jobs?: SemaphoreJobEntry[];
};

type SemaphoreBlocksValue = {
  __type: "semaphoreBlocks";
  blocks: SemaphoreBlockEntry[];
};

type PagerDutyIncidentEntry = {
  id: string;
  title: string;
  status: string;
  urgency: string;
  service?: string;
  priority?: string;
  html_url?: string;
  created_at?: string;
};

interface ChainItemProps {
  item: ChainItemData;
  index: number;
  totalItems?: number;
  isOpen: boolean;
  isSelected?: boolean;
  onToggleOpen: (itemId: string) => void;
  getExecutionState?: (
    nodeId: string,
    execution: CanvasesCanvasNodeExecution,
  ) => { map: EventStateMap; state: EventState };
}

export const ChainItem: React.FC<ChainItemProps> = ({
  item,
  index,
  totalItems,
  isOpen,
  isSelected = false,
  onToggleOpen,
  getExecutionState,
}) => {
  const [activeTab, setActiveTab] = useState<"current" | "payload">("current");
  const [isPayloadModalOpen, setIsPayloadModalOpen] = useState(false);
  const [modalPayload, setModalPayload] = useState<any>(null);
  const [payloadCopied, setPayloadCopied] = useState(false);
  const state = useMemo(() => {
    if (!getExecutionState || !item.originalExecution) return item.state;
    const { state } = getExecutionState(item.nodeId || "", item.originalExecution);
    return state;
  }, [item.nodeId, item.originalExecution, getExecutionState, item.state]);

  const eventStateStyle: EventStateStyle = useMemo(() => {
    if (!getExecutionState || !item.originalExecution) {
      // Fallback to provided state or neutral
      const fallbackState = (item.state as EventState) || "neutral";
      return DEFAULT_EVENT_STATE_MAP[fallbackState] || DEFAULT_EVENT_STATE_MAP["neutral"];
    }

    const { map, state } = getExecutionState(item.nodeId || "", item.originalExecution);
    return map[state];
  }, [item.nodeId, item.originalExecution, getExecutionState, item.state]);

  const componentSubtitle = useMemo(() => {
    if (!item.workflowNode?.component?.name || !item.originalExecution) {
      return undefined;
    }

    const mapper = getComponentBaseMapper(item.workflowNode.component.name);

    // Pass a marker to indicate this is from ChainItem, so subtitle can skip issue counts
    const subtitle = mapper.subtitle?.({
      node: buildNodeInfo(item.workflowNode),
      execution: buildExecutionInfo(item.originalExecution),
      additionalData: { skipIssueCounts: true },
    });

    const parts = subtitle ? subtitle.toString().split(" · ") : [];
    if (parts.length > 1) {
      return parts[0];
    }

    return "";
  }, [item.workflowNode, item.originalExecution]);

  const copyToClipboard = useCallback((text: string) => {
    navigator.clipboard.writeText(text);
  }, []);

  const copyPayloadToClipboard = useCallback(
    (payload: any) => {
      const payloadString = typeof payload === "string" ? payload : JSON.stringify(payload, null, 2);
      copyToClipboard(payloadString);
      setPayloadCopied(true);
      setTimeout(() => setPayloadCopied(false), 2000);
    },
    [copyToClipboard],
  );

  const EventBackground = eventStateStyle.backgroundColor;
  const EventBadgeColor = eventStateStyle.badgeColor;

  const showConnectingLine = totalItems && index < totalItems - 1;
  const isDetailValue = (value: unknown): value is DetailValue => {
    if (!value || typeof value !== "object") return false;
    return "text" in value && typeof (value as DetailValue).text === "string";
  };
  const isExpressionBadges = (value: unknown): value is ExpressionBadges => {
    if (!value || typeof value !== "object") return false;
    return "__type" in value && (value as ExpressionBadges).__type === "expressionBadges";
  };
  const isEvaluationBadges = (value: unknown): value is EvaluationBadges => {
    if (!value || typeof value !== "object") return false;
    return "__type" in value && (value as EvaluationBadges).__type === "evaluationBadges";
  };
  const isErrorValue = (value: unknown): value is ErrorValue => {
    if (!value || typeof value !== "object") return false;
    return "__type" in value && (value as ErrorValue).__type === "error";
  };
  const isApprovalTimeline = (value: unknown): value is ApprovalTimelineEntry[] => {
    if (!Array.isArray(value)) return false;
    return value.every(
      (entry) =>
        entry &&
        typeof entry === "object" &&
        "label" in entry &&
        "status" in entry &&
        typeof (entry as ApprovalTimelineEntry).label === "string" &&
        typeof (entry as ApprovalTimelineEntry).status === "string",
    );
  };
  const isIssuesList = (value: unknown): value is IssueListEntry[] => {
    if (!Array.isArray(value)) return false;
    return value.every(
      (entry) =>
        entry &&
        typeof entry === "object" &&
        "status" in entry &&
        ((entry as IssueListEntry).status === "degraded" || (entry as IssueListEntry).status === "critical") &&
        "checkName" in entry &&
        typeof (entry as IssueListEntry).checkName === "string",
    );
  };
  const isSemaphoreBlocks = (value: unknown): value is SemaphoreBlocksValue => {
    if (!value || typeof value !== "object") return false;
    return "__type" in value && (value as SemaphoreBlocksValue).__type === "semaphoreBlocks";
  };
  const isPagerDutyIncidentsList = (value: unknown): value is PagerDutyIncidentEntry[] => {
    if (!Array.isArray(value)) return false;
    if (value.length === 0) return false;
    return value.every(
      (entry) =>
        entry &&
        typeof entry === "object" &&
        "id" in entry &&
        "title" in entry &&
        "status" in entry &&
        "urgency" in entry &&
        typeof (entry as PagerDutyIncidentEntry).id === "string" &&
        typeof (entry as PagerDutyIncidentEntry).title === "string" &&
        ((entry as PagerDutyIncidentEntry).status === "triggered" ||
          (entry as PagerDutyIncidentEntry).status === "acknowledged" ||
          (entry as PagerDutyIncidentEntry).status === "resolved"),
    );
  };
  const getUrgencyDotColor = (urgency: string) => {
    if (urgency === "high") return "bg-red-500";
    return "bg-yellow-500";
  };
  const getApprovalStatusColor = (status: string) => {
    const normalized = status.toLowerCase();
    if (normalized === "approved") return "bg-emerald-500";
    if (normalized === "rejected") return "bg-red-500";
    if (normalized === "critical") return "bg-red-500";
    if (normalized === "degraded") return "bg-yellow-500";
    return "bg-gray-400";
  };

  return (
    <div className="relative">
      <div
        key={item.id + index}
        className={
          `cursor-pointer p-2 relative rounded-md border-1 border-slate-950/20 ${
            isSelected ? "ring-[3px] ring-sky-300 ring-offset-3" : ""
          } ${EventBackground}` + (showConnectingLine ? " mb-3" : "")
        }
        onClick={(e) => {
          e.stopPropagation();
          onToggleOpen(item.id);
        }}
      >
        {/* First row: Component icon/name and state badge */}
        <div className="flex items-center justify-between gap-2 min-w-0 flex-1">
          <div className="flex items-center gap-2 min-w-0 flex-1">
            {/* Component Icon */}
            {(item.nodeIconSlug || item.nodeIcon) && (
              <div className="flex-shrink-0 w-4 h-4 flex items-center justify-center">
                {React.createElement(resolveIcon(item.nodeIconSlug || item.nodeIcon), {
                  size: 16,
                  className: "text-gray-800",
                })}
              </div>
            )}
            <span className="text-sm text-gray-800 truncate min-w-0 font-semibold">
              {item.nodeDisplayName || item.nodeName || item.componentName}
            </span>
          </div>
          <div className="flex items-center gap-2">
            {/* Component subtitle */}
            {componentSubtitle && <span className="text-sm text-gray-500 truncate">{componentSubtitle}</span>}
            <div
              className={`uppercase text-[11px] py-[1.5px] px-[5px] font-semibold rounded flex items-center tracking-wide justify-center text-white ${EventBadgeColor}`}
            >
              <span>{eventStateStyle.label || state}</span>
            </div>
          </div>
        </div>

        {/* Second row: Time ago and duration */}
        <div className="flex items-center mt-0 ml-6 gap-2">
          <span className="text-[13px] text-gray-950/60">
            {formatTimeAgo(new Date(item.originalExecution?.createdAt || item.originalEvent?.createdAt || ""))}
            {item.originalExecution?.state === "STATE_FINISHED" &&
              item.originalExecution?.createdAt &&
              item.originalExecution?.updatedAt && (
                <>
                  <span className="mx-1">·</span>
                  <span>
                    Duration:{" "}
                    {calcRelativeTimeFromDiff(
                      new Date(item.originalExecution.updatedAt).getTime() -
                        new Date(item.originalExecution.createdAt).getTime(),
                    )}
                  </span>
                </>
              )}
            {(item.childExecutions?.length || 0) > 0 && (
              <>
                <span className="mx-1">·</span>
                <span>
                  {item.childExecutions?.length} execution{item.childExecutions!.length > 1 ? "s" : ""}
                </span>
              </>
            )}
          </span>
        </div>

        {/* Child executions for composite components */}
        {item.childExecutions && item.childExecutions.length > 0 && (
          <div className="ml-8 mt-1 space-y-1">
            {item.childExecutions.map((child, childIndex) => (
              <div key={`${item.id}-child-${childIndex}`} className="flex items-center justify-between gap-2 text-sm">
                <div className="flex items-center gap-2 min-w-0 flex-1">
                  <div className="flex-shrink-0">
                    {React.createElement(resolveIcon("corner-down-right"), {
                      size: 16,
                      className: "text-gray-400",
                    })}
                  </div>
                  {/* Component Icon */}
                  {child.componentIcon && (
                    <div className="flex-shrink-0 w-4 h-4 flex items-center justify-center">
                      {React.createElement(resolveIcon(child.componentIcon), {
                        size: 14,
                        className: "text-gray-500",
                      })}
                    </div>
                  )}
                  <span className="text-sm text-gray-500 truncate flex-1">{child.name}</span>
                </div>
                <div
                  className={`capitalize text-xs py-[1px] px-[3px] rounded flex items-center justify-center flex-shrink-0 ${
                    child.badgeColor?.replace("bg", "text") || "bg-gray-400"
                  }`}
                >
                  <span>{child.state}</span>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Expandable content */}
        {isOpen && item.tabData && (
          <div
            className="mt-3 ml-7 rounded-sm bg-white outline outline-slate-950/20 text-gray-500 w-[calc(100%-2rem)] mb-1"
            onClick={(e) => e.stopPropagation()}
          >
            {/* Tab Navigation */}
            <div className="flex items-center h-8 border-b-1 border-gray-300">
              <div className="flex">
                {item.tabData.current && (
                  <button
                    onClick={() => setActiveTab("current")}
                    className={`py-1.5 ml-4 text-[13px] font-medium rounded-tr-md flex items-center border-b-1 gap-1  ${
                      activeTab === "current"
                        ? "text-gray-800 border-b-1 border-gray-800"
                        : "text-gray-500 hover:text-gray-800"
                    }`}
                  >
                    {React.createElement(resolveIcon("Croissant"), { size: 16 })}
                    Details
                  </button>
                )}
              </div>
              {item.tabData.payload && (
                <button
                  onClick={() => setActiveTab("payload")}
                  className={`py-1.5 ml-4 text-[13px] font-medium rounded-tr-md flex items-center border-b-1 gap-1 ${
                    activeTab === "payload"
                      ? "text-gray-800 border-b-1 border-gray-800"
                      : "text-gray-500 hover:text-gray-800"
                  }`}
                >
                  {React.createElement(resolveIcon("code"), { size: 16 })}
                  Payload
                </button>
              )}
            </div>

            {/* Tab Content */}
            {activeTab === "current" && item.tabData.current && (
              <div className="w-full flex flex-col gap-1 items-center justify-between my-1 px-2 pt-2 pb-3">
                {Object.entries(item.tabData.current).map(([key, value]) => {
                  if (isApprovalTimeline(value)) {
                    return (
                      <div key={key} className="flex items-start gap-1 px-2 rounded-md w-full min-w-0 font-medium">
                        <span className="text-[13px] flex-shrink-0 text-right w-[30%] truncate" title={key}>
                          {key}:
                        </span>
                        <div className="text-[13px] flex-1 text-left w-[70%] text-gray-800 min-w-0">
                          <div className="flex flex-col gap-3">
                            {value.map((entry, entryIndex) => (
                              <div key={`${entry.label}-${entryIndex}`} className="relative pl-4">
                                <div
                                  className={`absolute left-0 top-1.5 h-2 w-2 rounded-full ${getApprovalStatusColor(
                                    entry.status,
                                  )}`}
                                />
                                {entryIndex < value.length - 1 && (
                                  <div className="absolute left-[3px] top-4 bottom-[-12px] w-px bg-gray-200" />
                                )}
                                {entry.label.includes(" · ") ? (
                                  // Handle combined label with status (e.g., "Check Name · STATUS")
                                  // Status is in label, so we don't show the separate status line
                                  <div className="text-[13px] text-gray-800 font-medium truncate" title={entry.label}>
                                    {entry.label.split(" · ").map((part, idx) => (
                                      <span key={idx}>
                                        {idx === 0 ? (
                                          <span>{part}</span>
                                        ) : (
                                          <span>
                                            {" · "}
                                            <span className="text-[12px] text-gray-600 font-normal">{part}</span>
                                          </span>
                                        )}
                                      </span>
                                    ))}
                                  </div>
                                ) : (
                                  <>
                                    <div className="text-[13px] text-gray-800 font-medium truncate" title={entry.label}>
                                      {entry.label}
                                    </div>
                                    {entry.status && (
                                      <div
                                        className="text-[12px] text-gray-600 truncate"
                                        title={`${entry.status}${entry.timestamp ? ` ${entry.timestamp}` : ""}`}
                                      >
                                        {entry.status}
                                        {entry.timestamp ? ` ${entry.timestamp}` : ""}
                                      </div>
                                    )}
                                  </>
                                )}
                                {entry.comment && (
                                  <div className="text-[12px] text-gray-500 italic break-words">"{entry.comment}"</div>
                                )}
                              </div>
                            ))}
                          </div>
                        </div>
                      </div>
                    );
                  }

                  if (isIssuesList(value)) {
                    return (
                      <div key={key} className="flex items-start gap-1 px-2 rounded-md w-full min-w-0 font-medium">
                        <span className="text-[13px] flex-shrink-0 text-right w-[30%] truncate" title={key}>
                          {key}:
                        </span>
                        <div className="text-[13px] flex-1 text-left w-[70%] text-gray-800 min-w-0">
                          <div className="flex flex-col gap-4">
                            {value.map((issue, issueIndex) => (
                              <div key={`${issue.checkName}-${issueIndex}`} className="flex flex-col">
                                <div className="flex items-start gap-2">
                                  {/* Status badge replaces the dot */}
                                  <span
                                    className={`text-xs font-medium px-1 py-0.5 rounded flex-shrink-0 uppercase leading-tight self-start ${
                                      issue.status === "critical"
                                        ? "bg-red-100 text-red-700"
                                        : "bg-yellow-100 text-yellow-700"
                                    }`}
                                  >
                                    {issue.status}
                                  </span>

                                  <div className="flex-1 min-w-0">
                                    {/* Check name */}
                                    <div className="mb-1">
                                      <span
                                        className="text-[13px] font-semibold text-gray-900 break-words"
                                        title={issue.checkName}
                                      >
                                        {issue.checkName}
                                      </span>
                                    </div>
                                  </div>
                                </div>

                                {/* Check summary - spans full width below badge */}
                                {issue.checkSummary && (
                                  <div
                                    className="text-[12px] text-gray-700 break-words mt-1 w-full"
                                    title={issue.checkSummary}
                                  >
                                    {issue.checkSummary}
                                  </div>
                                )}

                                {/* Check description - spans full width below badge */}
                                {issue.checkDescription && (
                                  <div
                                    className="text-[12px] text-gray-500 italic break-words mt-1 w-full"
                                    title={issue.checkDescription}
                                  >
                                    {issue.checkDescription}
                                  </div>
                                )}
                              </div>
                            ))}
                          </div>
                        </div>
                      </div>
                    );
                  }

                  if (isPagerDutyIncidentsList(value)) {
                    return (
                      <div key={key} className="flex items-start gap-1 px-2 rounded-md w-full min-w-0 font-medium">
                        <span className="text-[13px] flex-shrink-0 text-right w-[30%] truncate" title={key}>
                          {key}:
                        </span>
                        <div className="text-[13px] flex-1 text-left w-[70%] text-gray-800 min-w-0">
                          {value.length === 0 ? (
                            <span className="text-gray-500 italic">No incidents</span>
                          ) : (
                            <div className="flex flex-col gap-3">
                              {value.map((incident, incidentIndex) => (
                                <div key={`${incident.id}-${incidentIndex}`} className="relative pl-4">
                                  {/* Timeline dot - colored by urgency */}
                                  <div
                                    className={`absolute left-0 top-1.5 h-2 w-2 rounded-full ${getUrgencyDotColor(incident.urgency)}`}
                                  />
                                  {/* Timeline connecting line */}
                                  {incidentIndex < value.length - 1 && (
                                    <div className="absolute left-[3px] top-4 bottom-[-12px] w-px bg-gray-200" />
                                  )}

                                  {/* Incident title with link */}
                                  <div className="text-[13px] text-gray-800 font-medium">
                                    {incident.html_url ? (
                                      <a
                                        href={incident.html_url}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        className="break-words"
                                        style={{ textDecoration: "underline 1px" }}
                                        title={incident.title}
                                        onClick={(e) => e.stopPropagation()}
                                      >
                                        {incident.title}
                                      </a>
                                    ) : (
                                      <span className="break-words" title={incident.title}>
                                        {incident.title}
                                      </span>
                                    )}
                                    {incident.created_at && (
                                      <>
                                        {" · "}
                                        <span className="text-[12px] font-normal text-gray-500">
                                          {formatTimeAgo(new Date(incident.created_at))}
                                        </span>
                                      </>
                                    )}
                                  </div>

                                  {/* Service, status, and priority info */}
                                  <div className="text-[12px] text-gray-600 truncate">
                                    <span className="capitalize">{incident.status}</span>
                                    {incident.service && (
                                      <>
                                        {" · "}
                                        <span title={incident.service}>{incident.service}</span>
                                      </>
                                    )}
                                    {incident.priority && (
                                      <>
                                        {" · "}
                                        <span title={`Priority: ${incident.priority}`}>{incident.priority}</span>
                                      </>
                                    )}
                                  </div>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      </div>
                    );
                  }

                  if (isSemaphoreBlocks(value)) {
                    return (
                      <div key={key} className="flex items-start gap-1 px-2 rounded-md w-full min-w-0 font-medium">
                        <span className="text-[13px] flex-shrink-0 text-right w-[30%] truncate" title={key}>
                          {key}:
                        </span>
                        <div className="text-[13px] flex-1 text-left w-[70%] text-gray-800 min-w-0">
                          <div className="flex flex-col gap-3">
                            {value.blocks.map((block, blockIndex) => {
                              const blockTitle = block.name || `Block ${blockIndex + 1}`;
                              const blockStatusParts = [block.result, block.state, block.resultReason].filter(Boolean);
                              return (
                                <div key={`${blockTitle}-${blockIndex}`} className="flex flex-col gap-1">
                                  <div className="text-[13px] text-gray-800 font-medium truncate" title={blockTitle}>
                                    {blockTitle}
                                    {blockStatusParts.length > 0 && (
                                      <span className="text-[12px] text-gray-600 font-normal">
                                        {" "}
                                        · {blockStatusParts.join(" · ")}
                                      </span>
                                    )}
                                  </div>
                                  {(block.jobs || []).length > 0 && (
                                    <div className="flex flex-col gap-1 pl-2">
                                      {block.jobs!.map((job, jobIndex) => {
                                        const jobTitle = job.name || `Job ${jobIndex + 1}`;
                                        const jobStatusParts = [job.result, job.status].filter(Boolean);
                                        return (
                                          <div
                                            key={`${jobTitle}-${jobIndex}`}
                                            className="text-[12px] text-gray-600 truncate"
                                            title={
                                              jobStatusParts.length > 0
                                                ? `${jobTitle} · ${jobStatusParts.join(" · ")}`
                                                : jobTitle
                                            }
                                          >
                                            {jobTitle}
                                            {jobStatusParts.length > 0 && (
                                              <span className="text-gray-500"> · {jobStatusParts.join(" · ")}</span>
                                            )}
                                          </div>
                                        );
                                      })}
                                    </div>
                                  )}
                                </div>
                              );
                            })}
                          </div>
                        </div>
                      </div>
                    );
                  }

                  if (isExpressionBadges(value)) {
                    return (
                      <div key={key} className="flex items-start gap-1 px-2 rounded-md w-full min-w-0 font-medium">
                        <span className="text-[13px] flex-shrink-0 text-right w-[30%] truncate" title={key}>
                          {key}:
                        </span>
                        <div className="text-[13px] flex-1 text-left w-[70%] text-gray-800 min-w-0 overflow-hidden">
                          <div className="flex flex-col gap-2 max-w-full">
                            {value.values.map((specValue, specIndex) => {
                              // Check if the last badge is a logical operator
                              const badges = specValue.badges;
                              const lastBadge = badges[badges.length - 1];
                              const isLogicalOperator =
                                lastBadge &&
                                (lastBadge.label === "&&" ||
                                  lastBadge.label === "||" ||
                                  lastBadge.label.toLowerCase() === "and" ||
                                  lastBadge.label.toLowerCase() === "or");

                              if (isLogicalOperator && badges.length > 1) {
                                // Split: render condition on one line, operator on its own line
                                return (
                                  <React.Fragment key={specIndex}>
                                    <div className="flex items-center gap-2 flex-wrap">
                                      {badges.slice(0, -1).map((badge, badgeIndex) => (
                                        <span
                                          key={badgeIndex}
                                          className={`px-2 py-1 rounded text-xs font-mono whitespace-nowrap flex-shrink-0 ${badge.bgColor} ${badge.textColor}`}
                                        >
                                          {badge.label}
                                        </span>
                                      ))}
                                    </div>
                                    <div className="flex items-center gap-2">
                                      <span
                                        className={`px-2 py-1 rounded text-xs font-mono whitespace-nowrap flex-shrink-0 ${lastBadge.bgColor} ${lastBadge.textColor}`}
                                      >
                                        {lastBadge.label}
                                      </span>
                                    </div>
                                  </React.Fragment>
                                );
                              }

                              // Normal rendering for groups without logical operators at the end
                              return (
                                <div key={specIndex} className="flex items-center gap-2 flex-wrap">
                                  {badges.map((badge, badgeIndex) => (
                                    <span
                                      key={badgeIndex}
                                      className={`px-2 py-1 rounded text-xs font-mono whitespace-nowrap flex-shrink-0 ${badge.bgColor} ${badge.textColor}`}
                                    >
                                      {badge.label}
                                    </span>
                                  ))}
                                </div>
                              );
                            })}
                          </div>
                        </div>
                      </div>
                    );
                  }

                  if (isEvaluationBadges(value)) {
                    // Operators that should always use default gray styling
                    const operators = new Set([
                      ">=",
                      "<=",
                      "==",
                      "!=",
                      ">",
                      "<",
                      "contains",
                      "startswith",
                      "endswith",
                      "matches",
                      "in",
                      "!",
                      "+",
                      "-",
                      "*",
                      "/",
                      "%",
                      "**",
                      "??",
                      "?",
                      ":",
                    ]);
                    const logicalOperators = new Set(["and", "or", "||", "&&"]);

                    // Helper to determine if a badge should be red
                    // A badge is red if it's part of a failed comparison (but not if it's an operator)
                    const shouldBeRed = (badgeLabel: string) => {
                      if (!value.failedParts || value.failedParts.length === 0) {
                        return false;
                      }
                      // Operators always use their default gray styling
                      const normalizedLabel = badgeLabel.trim().toLowerCase();
                      if (operators.has(normalizedLabel) || logicalOperators.has(normalizedLabel)) {
                        return false;
                      }
                      // Check if this badge label matches any failed part
                      const trimmedLabel = badgeLabel.trim();
                      return value.failedParts.some((failedPart) => {
                        const normalizedFailed = failedPart.trim();
                        return trimmedLabel === normalizedFailed;
                      });
                    };

                    return (
                      <div key={key} className="flex items-start gap-1 px-2 rounded-md w-full min-w-0 font-medium">
                        <span className="text-[13px] flex-shrink-0 text-right w-[30%] truncate" title={key}>
                          {key}:
                        </span>
                        <div className="text-[13px] flex-1 text-left w-[70%] text-gray-800 min-w-0 overflow-hidden">
                          <div className="flex flex-col gap-2 max-w-full">
                            {value.values.map((specValue, specIndex) => {
                              // Check if the last badge is a logical operator
                              const badges = specValue.badges;
                              const lastBadge = badges[badges.length - 1];
                              const isLogicalOperator =
                                lastBadge &&
                                (lastBadge.label === "&&" ||
                                  lastBadge.label === "||" ||
                                  lastBadge.label.toLowerCase() === "and" ||
                                  lastBadge.label.toLowerCase() === "or");

                              if (isLogicalOperator && badges.length > 1) {
                                // Split: render condition on one line, operator on its own line
                                return (
                                  <React.Fragment key={specIndex}>
                                    <div className="flex items-center gap-2 flex-wrap">
                                      {badges.slice(0, -1).map((badge, badgeIndex) => {
                                        // Use red only for null/undefined when failed, otherwise use original colors
                                        const isRed = shouldBeRed(badge.label);
                                        const badgeBg = isRed ? "bg-red-200" : badge.bgColor;
                                        const badgeText = isRed ? "text-red-900" : badge.textColor;
                                        return (
                                          <span
                                            key={badgeIndex}
                                            className={`px-2 py-1 rounded text-xs font-mono whitespace-nowrap flex-shrink-0 ${badgeBg} ${badgeText}`}
                                          >
                                            {badge.label}
                                          </span>
                                        );
                                      })}
                                    </div>
                                    <div className="flex items-center gap-2">
                                      <span
                                        className={`px-2 py-1 rounded text-xs font-mono whitespace-nowrap flex-shrink-0 ${lastBadge.bgColor} ${lastBadge.textColor}`}
                                      >
                                        {lastBadge.label}
                                      </span>
                                    </div>
                                  </React.Fragment>
                                );
                              }

                              // Normal rendering for groups without logical operators at the end
                              return (
                                <div key={specIndex} className="flex items-center gap-2 flex-wrap">
                                  {badges.map((badge, badgeIndex) => {
                                    // Use red only for null/undefined when failed, otherwise use original colors
                                    const isRed = shouldBeRed(badge.label);
                                    const badgeBg = isRed ? "bg-red-200" : badge.bgColor;
                                    const badgeText = isRed ? "text-red-900" : badge.textColor;
                                    return (
                                      <span
                                        key={badgeIndex}
                                        className={`px-2 py-1 rounded text-xs font-mono whitespace-nowrap flex-shrink-0 ${badgeBg} ${badgeText}`}
                                      >
                                        {badge.label}
                                      </span>
                                    );
                                  })}
                                </div>
                              );
                            })}
                          </div>
                        </div>
                      </div>
                    );
                  }

                  if (isErrorValue(value)) {
                    return (
                      <div key={key} className="flex items-start gap-1 px-2 rounded-md w-full min-w-0 font-medium">
                        <span
                          className="text-[13px] flex-shrink-0 text-right w-[30%] truncate text-red-600"
                          title={key}
                        >
                          {key}:
                        </span>
                        <div className="text-[13px] flex-1 text-left w-[70%] text-red-600 min-w-0">
                          <div className="break-words whitespace-normal" title={value.message}>
                            {value.message}
                          </div>
                        </div>
                      </div>
                    );
                  }

                  if (isDetailValue(value)) {
                    return (
                      <div key={key} className="flex items-start gap-1 px-2 rounded-md w-full min-w-0 font-medium">
                        <span className="text-[13px] flex-shrink-0 text-right w-[30%] truncate" title={key}>
                          {key}:
                        </span>
                        <div className="text-[13px] flex-1 text-left w-[70%] text-gray-800 min-w-0">
                          <div className="truncate" title={value.text}>
                            {value.text}
                          </div>
                          {value.comment && (
                            <div className="text-[12px] text-gray-500 italic truncate" title={value.comment}>
                              "{value.comment}"
                            </div>
                          )}
                        </div>
                      </div>
                    );
                  }

                  const stringValue = String(value);
                  const isUrlValue = isUrl(stringValue);

                  return (
                    <div key={key} className="flex items-center gap-1 px-2 rounded-md w-full min-w-0 font-medium">
                      <span className="text-[13px] flex-shrink-0 text-right w-[30%] truncate" title={key}>
                        {key}:
                      </span>
                      {isUrlValue ? (
                        <a
                          href={stringValue}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-[13px] flex-1 text-left w-[70%] text-gray-800 cursor-pointer inline-block overflow-hidden text-ellipsis whitespace-nowrap max-w-full"
                          style={{ textDecoration: "underline", textDecorationThickness: "1px" }}
                          title={stringValue}
                          onClick={(e) => e.stopPropagation()}
                        >
                          {stringValue}
                        </a>
                      ) : (
                        <span
                          className="text-[13px] flex-1 truncate text-left w-[70%] hover:underline text-gray-800 truncate"
                          title={stringValue}
                        >
                          {stringValue}
                        </span>
                      )}
                    </div>
                  );
                })}
              </div>
            )}

            {activeTab === "payload" && item.tabData.payload && (
              <div className="w-full">
                <div className="flex items-center justify-between mb-2 relative">
                  <div className="flex items-center gap-1 absolute right-1.5 top-1.5">
                    <SimpleTooltip content={payloadCopied ? "Copied!" : "Copy Link"} hideOnClick={false}>
                      <button
                        onClick={() => copyPayloadToClipboard(item.tabData!.payload)}
                        className="p-1 rounded text-gray-500 hover:text-gray-800"
                      >
                        {React.createElement(resolveIcon("copy"), { size: 14 })}
                      </button>
                    </SimpleTooltip>
                    <SimpleTooltip content="Payload">
                      <button
                        onClick={() => {
                          setModalPayload(item.tabData!.payload);
                          setIsPayloadModalOpen(true);
                        }}
                        className="p-1 text-gray-500 hover:text-gray-800"
                      >
                        {React.createElement(resolveIcon("maximize-2"), { size: 14 })}
                      </button>
                    </SimpleTooltip>
                  </div>
                </div>
                <div className="h-50 overflow-auto rounded -mt-2">
                  <JsonView
                    value={
                      typeof item.tabData.payload === "string" ? JSON.parse(item.tabData.payload) : item.tabData.payload
                    }
                    style={{
                      fontSize: "12px",
                      fontFamily:
                        'Monaco, Menlo, "Cascadia Code", "Segoe UI Mono", "Roboto Mono", Consolas, "Courier New", monospace',
                      backgroundColor: "#ffffff",
                      color: "#24292e",
                      padding: "8px",
                    }}
                    className="json-viewer-hide-types"
                    displayObjectSize={false}
                    enableClipboard={false}
                  />
                </div>
              </div>
            )}
          </div>
        )}

        {/* Payload Modal */}
        <Dialog
          open={isPayloadModalOpen}
          onOpenChange={(open) => {
            if (!open) {
              setIsPayloadModalOpen(false);
              setModalPayload(null);
            }
          }}
        >
          <DialogContent
            size="large"
            className="w-[80vw] max-w-[80vw] h-[80vh] max-h-[80vh] flex flex-col"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between">
              <DialogTitle>Payload</DialogTitle>
              <DialogDescription className="sr-only">Expanded payload viewer.</DialogDescription>
              <SimpleTooltip content={payloadCopied ? "Copied!" : "Copy"} hideOnClick={false}>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    copyPayloadToClipboard(modalPayload);
                  }}
                  className="px-3 py-1 text-sm text-gray-800 bg-gray-50 hover:bg-gray-200 rounded flex items-center gap-1"
                >
                  {React.createElement(resolveIcon("copy"), { size: 14 })}
                  Copy
                </button>
              </SimpleTooltip>
            </div>
            <div className="flex-1 overflow-auto border border-gray-200 dark:border-gray-700 rounded-md">
              <div className="p-4">
                {modalPayload && (
                  <JsonView
                    value={typeof modalPayload === "string" ? JSON.parse(modalPayload) : modalPayload}
                    style={{
                      fontSize: "14px",
                      fontFamily:
                        'Monaco, Menlo, "Cascadia Code", "Segoe UI Mono", "Roboto Mono", Consolas, "Courier New", monospace',
                      backgroundColor: "#ffffff",
                      color: "#24292e",
                    }}
                    className="json-viewer-hide-types"
                    displayObjectSize={false}
                    enableClipboard={false}
                  />
                )}
              </div>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {/* Connecting line */}
      {showConnectingLine && <div className="absolute left-5 -bottom-3 w-[1px] h-3 bg-slate-400 z-10" />}
    </div>
  );
};
