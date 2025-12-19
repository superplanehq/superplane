/* eslint-disable @typescript-eslint/no-explicit-any */
import { resolveIcon, isUrl } from "@/lib/utils";
import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { SidebarEvent } from "../types";
import { SidebarEventActionsMenu } from "./SidebarEventActionsMenu";
import JsonView from "@uiw/react-json-view";
import { SimpleTooltip } from "../SimpleTooltip";
import { DEFAULT_EVENT_STATE_MAP, EventState, EventStateMap, EventStateStyle } from "@/ui/componentBase";
import { WorkflowsWorkflowNodeExecution } from "@/api-client";

export interface ExecutionChainItem extends EventStateStyle {
  name: string;
  nodeId: string;
  executionId: string;
  state: string;
  payload?: any;
  children?: Array<{ name: string; state: string } & EventStateStyle>;
}

export interface TabData {
  current?: Record<string, any>;
  root?: Record<string, any>;
  payload?: any;
  executionChain?: ExecutionChainItem[];
}

interface SidebarEventItemProps {
  event: SidebarEvent;
  index: number;
  totalItems?: number;
  variant?: "latest" | "queue";
  isOpen: boolean;
  onToggleOpen: (eventId: string) => void;
  onEventClick?: (event: SidebarEvent) => void;
  onTriggerNavigate?: (event: SidebarEvent) => void;
  tabData?: TabData;
  onCancelQueueItem?: (id: string) => void;
  onCancelExecution?: (executionId: string) => void;
  onPushThrough?: (executionId: string) => void;
  supportsPushThrough?: boolean;
  onReEmit?: (nodeId: string, eventOrExecutionId: string) => void;
  loadExecutionChain?: (
    eventId: string,
    nodeId?: string,
    currentExecution?: Record<string, unknown>,
    forceReload?: boolean,
  ) => Promise<any[]>;
  getExecutionState?: (
    nodeId: string,
    execution: WorkflowsWorkflowNodeExecution,
  ) => { map: EventStateMap; state: EventState };
}

export const SidebarEventItem: React.FC<SidebarEventItemProps> = ({
  event,
  index,
  totalItems,
  isOpen,
  onToggleOpen,
  onEventClick,
  onTriggerNavigate,
  tabData,
  onCancelQueueItem,
  onCancelExecution,
  onPushThrough,
  supportsPushThrough,
  onReEmit,
  loadExecutionChain,
  getExecutionState,
}) => {
  // Determine default active tab based on available data
  const getDefaultActiveTab = useCallback((): "current" | "root" | "payload" | "executionChain" => {
    if (!tabData) return "current";
    if (tabData.current) return "current";
    if (tabData.root) return "root";
    if (tabData.payload) return "payload";
    // Execution chain will be loaded lazily, so don't default to it
    return "current";
  }, [tabData]);

  const [activeTab, setActiveTab] = useState<"current" | "root" | "payload" | "executionChain">(getDefaultActiveTab());
  const [isPayloadModalOpen, setIsPayloadModalOpen] = useState(false);
  const [modalPayload, setModalPayload] = useState<any>(null);
  const [copiedExecutions, setCopiedExecutions] = useState<Set<string>>(new Set());
  const [payloadCopied, setPayloadCopied] = useState(false);
  const [executionChainData, setExecutionChainData] = useState<ExecutionChainItem[] | null>(null);
  const [executionChainLoading, setExecutionChainLoading] = useState(false);

  const eventStateStyle: EventStateStyle = useMemo(() => {
    if (!getExecutionState) return DEFAULT_EVENT_STATE_MAP["neutral"];

    if (event.kind === "queue") return DEFAULT_EVENT_STATE_MAP["queued"];

    if (event.kind === "trigger") {
      return DEFAULT_EVENT_STATE_MAP[event.state as EventState] || DEFAULT_EVENT_STATE_MAP["neutral"];
    }

    const { map, state } = getExecutionState(
      event.nodeId || "",
      event.originalExecution as WorkflowsWorkflowNodeExecution,
    );
    return map[state];
  }, [event.nodeId, event.originalExecution, getExecutionState, event.kind, event.state]);

  // Function to load execution chain data lazily
  const loadExecutionChainData = useCallback(
    async (forceReload = false) => {
      if (!loadExecutionChain || executionChainLoading) return;

      if (executionChainData && !forceReload) return;

      const rootEventId = tabData?.root?.["Event ID"];
      if (!rootEventId || typeof rootEventId !== "string") return;

      try {
        if (!forceReload) {
          setExecutionChainLoading(true);
        }
        const currentNodeId = event.nodeId || tabData?.current?.["Node ID"] || "";
        const currentExecution = tabData?.current;

        const rawExecutionChain = await loadExecutionChain(rootEventId, currentNodeId, currentExecution, forceReload);

        const processedChainData = rawExecutionChain.map((exec: any) => {
          if (!getExecutionState) return {};

          const { map, state } = getExecutionState(exec.nodeId, exec);
          const eventStyle = map[state];

          let payload: Record<string, unknown> = {};
          if (exec.outputs) {
            const outputData: unknown[] = Object.values(exec.outputs)?.find((output) => {
              return Array.isArray(output) && output?.length > 0;
            }) as unknown[];

            if (outputData?.length > 0) {
              const output = outputData?.[0] as Record<string, unknown>;
              if (output["data"]) {
                payload = (output["data"] as Record<string, unknown>) || {};
              } else {
                payload = output || {};
              }
            }
          }

          const mainItem: ExecutionChainItem = {
            ...eventStyle,
            name: exec.nodeId || "Unknown",
            nodeId: exec.nodeId || "",
            executionId: exec.id || "",
            payload,
            state: state,
            children:
              exec?.childExecutions && exec.childExecutions.length > 0
                ? exec.childExecutions
                    .slice()
                    .sort((a: any, b: any) => {
                      const timeA = a.createdAt ? new Date(a.createdAt).getTime() : 0;
                      const timeB = b.createdAt ? new Date(b.createdAt).getTime() : 0;
                      return timeA - timeB;
                    })
                    .map((childExec: any) => {
                      const nodeId = childExec?.nodeId?.split(":")?.at(-1);
                      const { map, state } = getExecutionState(exec.nodeId, childExec);

                      return {
                        name: nodeId || "Unknown",
                        state: state,
                        ...map[state],
                      };
                    })
                : undefined,
          };

          return mainItem;
        }) as ExecutionChainItem[];

        setExecutionChainData(processedChainData);
      } catch (error) {
        console.error("Failed to load execution chain:", error);
        if (!forceReload) {
          setExecutionChainData([]);
        }
      } finally {
        if (!forceReload) {
          setExecutionChainLoading(false);
        }
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [loadExecutionChain, tabData, event.nodeId],
  );

  const navigate = useNavigate();

  // Use ref to track current values without causing re-renders
  const pollingRef = useRef<{
    activeTab: string;
    hasInProgress: boolean;
    loadData: (() => void) | null;
  }>({
    activeTab,
    hasInProgress: executionChainData?.some((item) => item.state === "running") || false,
    loadData: null,
  });

  pollingRef.current.activeTab = activeTab;
  pollingRef.current.hasInProgress =
    ["waiting", "running", "pending"].includes(event.state || "") ||
    executionChainData?.some((item) => item.state !== "running" && item.state !== "failed") ||
    executionChainData?.some((execution) =>
      execution.children?.some((children) => children.state !== "success" && children.state !== "failed"),
    ) ||
    false;
  pollingRef.current.loadData = () => loadExecutionChainData(true);

  useEffect(() => {
    const pollInterval = setInterval(() => {
      const { activeTab: currentTab, hasInProgress, loadData } = pollingRef.current;

      if (currentTab === "executionChain" && hasInProgress && loadData) {
        loadData();
      }
    }, 1500);

    return () => {
      clearInterval(pollInterval);
    };
  }, []);

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

  const copyExecutionLink = useCallback(
    (execution: ExecutionChainItem) => {
      const pathParts = window.location.pathname.split("/");
      const orgId = pathParts[1];
      const workflowId = pathParts[3];

      if ((execution.children?.length || 0) > 0) {
        const nodeId = execution.nodeId;
        const executionId = execution.executionId;

        const link = `${window.location.origin}/${orgId}/workflows/${workflowId}/nodes/${nodeId}/${executionId}`;
        copyToClipboard(link);
      } else {
        const link = `${window.location.origin}/${orgId}/workflows/${workflowId}?sidebar=1&node=${execution.nodeId}`;
        copyToClipboard(link);
      }

      const executionKey = `${execution.nodeId}-${execution.executionId}`;
      setCopiedExecutions((prev) => new Set(prev).add(executionKey));
      setTimeout(() => {
        setCopiedExecutions((prev) => {
          const newSet = new Set(prev);
          newSet.delete(executionKey);
          return newSet;
        });
      }, 2000);
    },
    [copyToClipboard],
  );

  const handleExpandCustomComponentExecution = useCallback(
    (execution: ExecutionChainItem) => {
      const pathParts = window.location.pathname.split("/");
      const orgId = pathParts[1];
      const workflowId = pathParts[3];

      const nodeId = execution.nodeId;
      const executionId = execution.executionId;

      const path = `/${orgId}/workflows/${workflowId}/nodes/${nodeId}/${executionId}`;
      navigate(path, { replace: false });
    },
    [navigate],
  );

  const showExecutionPayload = useCallback(
    (execution: ExecutionChainItem) => {
      const payload = execution.payload || tabData?.payload;
      if (payload) {
        setModalPayload(payload);
        setIsPayloadModalOpen(true);
      }
    },
    [tabData?.payload],
  );

  // Update active tab when tabData changes to ensure we always have a valid active tab
  useEffect(() => {
    const defaultTab = getDefaultActiveTab();
    // Only update if current active tab is not available in the new tabData
    if (tabData) {
      if (activeTab === "current" && !tabData.current) {
        setActiveTab(defaultTab);
      } else if (activeTab === "root" && !tabData.root) {
        setActiveTab(defaultTab);
      } else if (activeTab === "payload" && !tabData.payload) {
        setActiveTab(defaultTab);
      }
      // For execution chain, don't auto-switch away since it's loaded on demand
    }
  }, [tabData, activeTab, getDefaultActiveTab]);

  const EventBackground = eventStateStyle.backgroundColor;
  const EventBadgeColor = eventStateStyle.badgeColor;

  return (
    <div
      key={event.title + index}
      className={
        `cursor-pointer px-4 pt-2 pb-3 relative rounded-lg border-1 border-slate-400 ${EventBackground}` +
        (totalItems && index < totalItems - 1 ? " mb-3" : "")
      }
      onClick={(e) => {
        e.stopPropagation();
        // For trigger events and component executions, navigate to execution chain instead of toggling inline
        if ((event.kind === "trigger" || event.kind === "execution") && onTriggerNavigate) {
          onTriggerNavigate(event);
        } else {
          onToggleOpen(event.id);
        }
        onEventClick?.(event);
      }}
    >
      {/* First row: Badge and subtitle */}
      <div className="flex items-center justify-between gap-2 min-w-0 flex-1">
        <div
          className={`uppercase text-sm py-[1px] px-[6px] font-semibold rounded flex items-center justify-center text-white ${EventBadgeColor}`}
        >
          <span>{event.state || "neutral"}</span>
        </div>
        {event.subtitle && (
          <span className="text-sm truncate flex-shrink-0 max-w-[40%] text-gray-500">{event.subtitle}</span>
        )}
      </div>

      {/* Second row: Event ID and title with actions */}
      <div className="flex items-center mt-2 gap-2">
        <div className="flex items-center gap-2 min-w-0 flex-1 cursor-pointer">
          {event.id && <span className="text-[13px] text-gray-950/50 font-mono">#{event.id?.slice(0, 4)}</span>}
          <span className="text-sm text-gray-700 font-inter truncate text-md min-w-0 font-semibold">{event.title}</span>
        </div>

        <div onClick={(e) => e.stopPropagation()}>
          <SidebarEventActionsMenu
            eventId={event.id}
            executionId={event.executionId}
            onCancelQueueItem={onCancelQueueItem}
            onCancelExecution={onCancelExecution}
            onPushThrough={onPushThrough}
            supportsPushThrough={supportsPushThrough}
            eventState={event.state}
            kind={event.kind || "execution"}
            onReEmit={() => {
              if (["queue", "execution"].includes(event.kind || "")) return;
              onReEmit?.(event.nodeId || "", event.id);
            }}
          />
        </div>
      </div>

      {isOpen && ((event.values && Object.entries(event.values).length > 0) || tabData) && (
        <div
          className="mt-3 rounded-sm bg-white outline outline-black/15 text-gray-500 w-full mb-0.5"
          onClick={(e) => e.stopPropagation()}
        >
          {/* Tab Navigation */}
          {tabData && (
            <div className="flex justify-between items-center border-b-1 border-gray-200">
              <div className="flex">
                {tabData.current && (
                  <button
                    onClick={() => setActiveTab("current")}
                    className={`px-5 py-1 text-sm font-medium rounded-tl-md  ${
                      activeTab === "current"
                        ? "text-black border-b-1 border-black"
                        : "text-gray-500 hover:text-gray-700 hover:bg-gray-50"
                    }`}
                  >
                    Current
                  </button>
                )}
                {tabData.root && (
                  <button
                    onClick={() => setActiveTab("root")}
                    className={`px-5 py-1 text-sm font-medium ${
                      activeTab === "root"
                        ? "text-black border-b-1 border-black"
                        : "text-gray-500 hover:text-gray-700 hover:bg-gray-50"
                    }`}
                  >
                    Root
                  </button>
                )}
                {(tabData?.executionChain || tabData?.root) && (
                  <button
                    onClick={() => {
                      setActiveTab("executionChain");
                      if (activeTab !== "executionChain") {
                        loadExecutionChainData();
                      }
                    }}
                    className={`px-5 py-1 text-sm font-medium ${
                      activeTab === "executionChain"
                        ? "text-black border-b-1 border-black"
                        : "text-gray-500 hover:text-gray-700 hover:bg-gray-50"
                    }`}
                  >
                    Execution Chain
                  </button>
                )}
              </div>
              {tabData.payload && (
                <button
                  onClick={() => setActiveTab("payload")}
                  className={`px-3 py-1 text-sm font-medium rounded-tr-md flex items-center gap-1 ${
                    activeTab === "payload"
                      ? "text-black border-b-1 border-black bg-gray-100"
                      : "text-gray-500 hover:text-gray-700 hover:bg-gray-50 border-l-1 border-gray-200"
                  }`}
                >
                  {React.createElement(resolveIcon("code"), { size: 16 })}
                  Payload
                </button>
              )}
            </div>
          )}

          {/* Tab Content */}
          {tabData && activeTab === "current" && tabData.current && (
            <div className="w-full flex flex-col gap-1 items-center justify-between my-1 px-2 py-2">
              {Object.entries(tabData.current).map(([key, value]) => {
                const stringValue = String(value);
                const isUrlValue = isUrl(stringValue);

                return (
                  <div key={key} className="flex items-center gap-1 px-2 rounded-md w-full min-w-0 font-medium">
                    <span className="text-sm flex-shrink-0 text-right w-[30%] truncate" title={key}>
                      {key}:
                    </span>
                    {isUrlValue ? (
                      <a
                        href={stringValue}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-sm flex-1 text-left w-[70%] text-gray-800 cursor-pointer inline-block overflow-hidden text-ellipsis whitespace-nowrap max-w-full"
                        style={{ textDecoration: "underline", textDecorationThickness: "1px" }}
                        title={stringValue}
                        onClick={(e) => e.stopPropagation()}
                      >
                        {stringValue}
                      </a>
                    ) : (
                      <span
                        className="text-sm flex-1 truncate text-left w-[70%] hover:underline text-gray-800 truncate"
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

          {tabData && activeTab === "root" && tabData.root && (
            <div className="w-full flex flex-col gap-1 items-center justify-between my-1 px-2 py-2">
              {Object.entries(tabData.root).map(([key, value]) => {
                const stringValue = String(value);
                const isUrlValue = isUrl(stringValue);

                return (
                  <div key={key} className="flex items-center gap-1 px-2 rounded-md w-full min-w-0 font-medium">
                    <span className="text-sm flex-shrink-0 text-right w-[30%] truncate" title={key}>
                      {key}:
                    </span>
                    {isUrlValue ? (
                      <a
                        href={stringValue}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-sm flex-1 text-left w-[70%] text-gray-800 cursor-pointer inline-block overflow-hidden text-ellipsis whitespace-nowrap max-w-full"
                        style={{ textDecoration: "underline", textDecorationThickness: "1px" }}
                        title={stringValue}
                        onClick={(e) => e.stopPropagation()}
                      >
                        {stringValue}
                      </a>
                    ) : (
                      <span
                        className="text-sm flex-1 truncate text-left w-[70%] hover:underline text-gray-800 truncate"
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

          {tabData && activeTab === "payload" && tabData.payload && (
            <div className="w-full px-2 py-2">
              <div className="flex items-center justify-between mb-2 relative">
                <div className="flex items-center gap-1 absolute right-2 top-4">
                  <SimpleTooltip content={payloadCopied ? "Copied!" : "Copy Link"} hideOnClick={false}>
                    <button
                      onClick={() => copyPayloadToClipboard(tabData.payload)}
                      className="p-1 hover:bg-gray-100 rounded text-gray-500 hover:text-gray-700"
                    >
                      {React.createElement(resolveIcon("copy"), { size: 16 })}
                    </button>
                  </SimpleTooltip>
                  <SimpleTooltip content="Payload">
                    <button
                      onClick={() => {
                        setModalPayload(tabData.payload);
                        setIsPayloadModalOpen(true);
                      }}
                      className="p-1 hover:bg-gray-100 rounded text-gray-500 hover:text-gray-700"
                    >
                      {React.createElement(resolveIcon("maximize-2"), { size: 16 })}
                    </button>
                  </SimpleTooltip>
                </div>
              </div>
              <div className="h-50 overflow-auto border rounded bg-white">
                <JsonView
                  value={typeof tabData.payload === "string" ? JSON.parse(tabData.payload) : tabData.payload}
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

          {activeTab === "executionChain" && (tabData?.root || executionChainData) && (
            <div className="w-full flex flex-col px-2 py-2">
              {executionChainLoading ? (
                <div className="flex items-center justify-center py-8">
                  <div className="text-sm text-gray-500">Loading execution chain...</div>
                </div>
              ) : executionChainData && executionChainData.length > 0 ? (
                <>
                  <div className="text-sm text-gray-500 ml-2">
                    {executionChainData.length} execution{executionChainData.length === 1 ? "" : "s"}
                  </div>
                  {executionChainData.map((execution, index) => (
                    <div key={index} className="flex flex-col">
                      {/* Main execution */}
                      <div className="flex items-center gap-2 px-2 py-1 rounded-md w-full min-w-0 group hover:bg-gray-100">
                        <div
                          className={`uppercase text-xs py-[1px] px-[4px] font-semibold rounded flex items-center justify-center text-white ${execution.badgeColor} flex-shrink-0`}
                        >
                          <span>{execution.state}</span>
                        </div>
                        <span className="text-sm text-gray-800 truncate flex-1">{execution.name}</span>
                        {/* Hover Icons */}
                        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                          {/* See Group (Expand/Collapse) */}
                          {execution.children && execution.children.length > 0 && (
                            <SimpleTooltip content="See Group">
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleExpandCustomComponentExecution(execution);
                                }}
                                className="p-1 rounded text-gray-500"
                              >
                                {React.createElement(resolveIcon("expand"), { size: 14 })}
                              </button>
                            </SimpleTooltip>
                          )}
                          {/* Copy Link */}
                          <SimpleTooltip
                            content={
                              copiedExecutions.has(`${execution.nodeId}-${execution.executionId}`)
                                ? "Copied!"
                                : "Copy Link"
                            }
                            hideOnClick={false}
                          >
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                copyExecutionLink(execution);
                              }}
                              className="p-1 rounded text-gray-500"
                            >
                              {React.createElement(resolveIcon("link"), { size: 14 })}
                            </button>
                          </SimpleTooltip>
                          {/* Payload */}
                          <SimpleTooltip content="Payload">
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                showExecutionPayload(execution);
                              }}
                              className="p-1 rounded text-gray-500"
                            >
                              {React.createElement(resolveIcon("code"), { size: 14 })}
                            </button>
                          </SimpleTooltip>
                        </div>
                      </div>
                      {/* Children executions */}
                      {execution.children &&
                        execution.children.map((child, childIndex) => (
                          <div
                            key={`${index}-${childIndex}`}
                            className="flex items-center gap-2 px-2 py-1 rounded-md w-full min-w-0"
                          >
                            <div className="flex-shrink-0">
                              {React.createElement(resolveIcon("corner-down-right"), {
                                size: 16,
                                className: "text-gray-400",
                              })}
                            </div>
                            <div
                              className={`uppercase text-xs py-[1px] px-[3px] font-semibold rounded flex items-center justify-center text-white flex-shrink-0 ${child.badgeColor}`}
                            >
                              <span>{child.state}</span>
                            </div>
                            <span className="text-sm text-gray-700 truncate flex-1">{child.name}</span>
                          </div>
                        ))}
                    </div>
                  ))}
                </>
              ) : (
                <div className="flex items-center justify-center py-8">
                  <div className="text-sm text-gray-500">No execution chain data available</div>
                </div>
              )}
            </div>
          )}

          {/* Fallback to original values display if no tabData */}
          {!tabData && event.values && Object.entries(event.values).length > 0 && (
            <div className="w-full flex flex-col gap-1 items-center justify-between my-1 px-2 py-2">
              {Object.entries(event.values || {}).map(([key, value]) => {
                const isUrlValue = isUrl(value);

                return (
                  <div key={key} className="flex items-center gap-1 px-2 rounded-md w-full min-w-0">
                    <span className="text-sm flex-shrink-0 text-right w-[30%] truncate" title={key}>
                      {key}:
                    </span>
                    {isUrlValue ? (
                      <a
                        href={value}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-sm flex-1 text-left w-[70%] text-gray-800 cursor-pointer inline-block overflow-hidden text-ellipsis whitespace-nowrap max-w-full"
                        style={{ textDecoration: "underline", textDecorationThickness: "1px" }}
                        title={value}
                        onClick={(e) => e.stopPropagation()}
                      >
                        {value}
                      </a>
                    ) : (
                      <span
                        className="text-sm flex-1 truncate text-left w-[70%] hover:underline text-gray-800 truncate"
                        title={value}
                      >
                        {value}
                      </span>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* Payload Modal */}
      {isPayloadModalOpen && modalPayload && (
        <div className="fixed inset-0 bg-black/25 z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-lg w-full max-w-4xl max-h-[90vh] flex flex-col">
            <div className="flex items-center justify-between p-4 border-b">
              <h3 className="text-lg font-semibold text-gray-900">Payload</h3>
              <div className="flex items-center gap-2">
                <SimpleTooltip content={payloadCopied ? "Copied!" : "Copy Link"} hideOnClick={false}>
                  <button
                    onClick={() => copyPayloadToClipboard(modalPayload)}
                    className="px-3 py-1 text-sm text-gray-800 bg-gray-50 hover:bg-gray-200 rounded flex items-center gap-1"
                  >
                    {React.createElement(resolveIcon("copy"), { size: 14 })}
                    Copy
                  </button>
                </SimpleTooltip>
                <button
                  onClick={() => {
                    setIsPayloadModalOpen(false);
                    setModalPayload(null);
                  }}
                  className="p-1 hover:bg-gray-100 rounded text-gray-500 hover:text-gray-700"
                >
                  {React.createElement(resolveIcon("x"), { size: 16 })}
                </button>
              </div>
            </div>
            <div className="flex-1 overflow-auto bg-white rounded-b-lg">
              <div className="p-4">
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
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
