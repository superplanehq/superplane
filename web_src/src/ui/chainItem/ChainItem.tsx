/* eslint-disable @typescript-eslint/no-explicit-any */
import { resolveIcon, isUrl, calcRelativeTimeFromDiff } from "@/lib/utils";
import React, { useCallback, useMemo, useState } from "react";
import { DEFAULT_EVENT_STATE_MAP, EventState, EventStateMap, EventStateStyle } from "@/ui/componentBase";
import { WorkflowsWorkflowNodeExecution, ComponentsNode, WorkflowsWorkflowEvent } from "@/api-client";
import JsonView from "@uiw/react-json-view";
import { SimpleTooltip } from "../componentSidebar/SimpleTooltip";
import { formatTimeAgo } from "@/utils/date";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { getComponentBaseMapper } from "@/pages/workflowv2/mappers";

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
  originalExecution?: WorkflowsWorkflowNodeExecution; // Add execution data
  originalEvent?: WorkflowsWorkflowEvent; // Add event data for trigger events
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

type ApprovalTimelineEntry = {
  label: string;
  status: string;
  timestamp?: string;
  comment?: string;
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
    execution: WorkflowsWorkflowNodeExecution,
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
    return mapper.subtitle?.(item.workflowNode, item.originalExecution, item.additionalData);
  }, [item.workflowNode, item.originalExecution, item.additionalData]);

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
  const getApprovalStatusColor = (status: string) => {
    const normalized = status.toLowerCase();
    if (normalized === "approved") return "bg-emerald-500";
    if (normalized === "rejected") return "bg-red-500";
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
              <span>{state}</span>
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
                                <div className="text-[13px] text-gray-800 font-medium truncate" title={entry.label}>
                                  {entry.label}
                                </div>
                                <div
                                  className="text-[12px] text-gray-600 truncate"
                                  title={`${entry.status}${entry.timestamp ? ` ${entry.timestamp}` : ""}`}
                                >
                                  {entry.status}
                                  {entry.timestamp ? ` ${entry.timestamp}` : ""}
                                </div>
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
          <DialogContent className="max-w-4xl max-h-[90vh] flex flex-col" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between">
              <DialogTitle>Payload</DialogTitle>
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
