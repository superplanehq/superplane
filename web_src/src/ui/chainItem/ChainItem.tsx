/* eslint-disable @typescript-eslint/no-explicit-any */
import { resolveIcon, isUrl, calcRelativeTimeFromDiff } from "@/lib/utils";
import React, { useCallback, useMemo, useState } from "react";
import type { EventState, EventStateMap, EventStateStyle } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import type { CanvasesCanvasNodeExecution, ComponentsNode, CanvasesCanvasEvent } from "@/api-client";
import JsonView from "@uiw/react-json-view";
import { SimpleTooltip } from "../componentSidebar/SimpleTooltip";
import { TimeAgo } from "@/components/TimeAgo";
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
  tabData?: {
    current?: Record<string, any>;
    payload?: any;
    configuration?: any;
  };
}

type DetailValue = {
  text: string;
  comment?: string;
};

type ErrorValue = {
  __type: "error";
  message: string;
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

function getReactNodeText(node: React.ReactNode): string {
  if (node === null || node === undefined || typeof node === "boolean") {
    return "";
  }

  if (typeof node === "string" || typeof node === "number") {
    return String(node);
  }

  if (Array.isArray(node)) {
    return node.map((child) => getReactNodeText(child)).join("");
  }

  if (React.isValidElement<{ children?: React.ReactNode }>(node)) {
    return getReactNodeText(node.props.children);
  }

  return "";
}

function getComponentSubtitlePrefix(subtitle: React.ReactNode): string {
  const subtitleText = getReactNodeText(subtitle);
  if (!subtitleText.trim()) {
    return "";
  }

  const [prefix] = subtitleText.split(" · ");
  if (!subtitleText.includes(" · ") || !prefix?.trim()) {
    return "";
  }

  return prefix.trim();
}

function escapeStringValuesForJsonView(value: unknown): unknown {
  if (typeof value === "string") {
    return JSON.stringify(value).slice(1, -1);
  }

  if (Array.isArray(value)) {
    return value.map((item) => escapeStringValuesForJsonView(item));
  }

  if (value && typeof value === "object") {
    return Object.fromEntries(Object.entries(value).map(([key, item]) => [key, escapeStringValuesForJsonView(item)]));
  }

  return value;
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
  const [activeTab, setActiveTab] = useState<"current" | "payload" | "configuration">("current");
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
    });

    return getComponentSubtitlePrefix(subtitle);
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
  const payloadPreview = useMemo(
    () => (item.tabData ? escapeStringValuesForJsonView(item.tabData.payload) : undefined),
    [item.tabData],
  );
  const configurationPreview = useMemo(
    () => (item.tabData?.configuration ? escapeStringValuesForJsonView(item.tabData.configuration) : undefined),
    [item.tabData],
  );
  const modalPayloadPreview = useMemo(() => escapeStringValuesForJsonView(modalPayload), [modalPayload]);

  const showConnectingLine = totalItems && index < totalItems - 1;
  const isDetailValue = (value: unknown): value is DetailValue => {
    if (!value || typeof value !== "object") return false;
    return "text" in value && typeof (value as DetailValue).text === "string";
  };
  const isErrorValue = (value: unknown): value is ErrorValue => {
    if (!value || typeof value !== "object") return false;
    return "__type" in value && (value as ErrorValue).__type === "error";
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
            <TimeAgo date={new Date(item.originalExecution?.createdAt || item.originalEvent?.createdAt || "")} />
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
              {item.tabData.configuration && Object.keys(item.tabData.configuration).length > 0 && (
                <button
                  onClick={() => setActiveTab("configuration")}
                  className={`py-1.5 ml-4 text-[13px] font-medium rounded-tr-md flex items-center border-b-1 gap-1 ${
                    activeTab === "configuration"
                      ? "text-gray-800 border-b-1 border-gray-800"
                      : "text-gray-500 hover:text-gray-800"
                  }`}
                >
                  {React.createElement(resolveIcon("settings"), { size: 16 })}
                  Config
                </button>
              )}
            </div>

            {/* Tab Content */}
            {activeTab === "current" && item.tabData.current && (
              <div className="w-full flex flex-col gap-1 items-center justify-between my-1 px-2 pt-2 pb-3">
                {Object.entries(item.tabData.current).map(([key, value]) => {
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
                    value={payloadPreview as Record<string, unknown>}
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

            {activeTab === "configuration" && configurationPreview && (
              <div className="w-full">
                <div className="flex items-center justify-between mb-2 relative">
                  <div className="flex items-center gap-1 absolute right-1.5 top-1.5">
                    <SimpleTooltip content={payloadCopied ? "Copied!" : "Copy"} hideOnClick={false}>
                      <button
                        onClick={() => copyPayloadToClipboard(item.tabData!.configuration)}
                        className="p-1 rounded text-gray-500 hover:text-gray-800"
                      >
                        {React.createElement(resolveIcon("copy"), { size: 14 })}
                      </button>
                    </SimpleTooltip>
                    <SimpleTooltip content="Configuration">
                      <button
                        onClick={() => {
                          setModalPayload(item.tabData!.configuration);
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
                    value={configurationPreview as Record<string, unknown>}
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
                    value={modalPayloadPreview as Record<string, unknown>}
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
