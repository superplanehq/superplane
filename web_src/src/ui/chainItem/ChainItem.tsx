/* eslint-disable @typescript-eslint/no-explicit-any */
import { resolveIcon, isUrl, calcRelativeTimeFromDiff } from "@/lib/utils";
import React, { useCallback, useMemo, useState } from "react";
import { DEFAULT_EVENT_STATE_MAP, EventState, EventStateMap, EventStateStyle } from "@/ui/componentBase";
import { WorkflowsWorkflowNodeExecution } from "@/api-client";
import JsonView from "@uiw/react-json-view";
import { SimpleTooltip } from "../componentSidebar/SimpleTooltip";
import { formatTimeAgo } from "@/utils/date";

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
  tabData?: {
    current?: Record<string, any>;
    payload?: any;
  };
}

interface ChainItemProps {
  item: ChainItemData;
  index: number;
  totalItems?: number;
  isOpen: boolean;
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

  return (
    <div className="relative">
      <div
        key={item.id + index}
        className={
          `cursor-pointer px-4 pt-2 pb-2 relative rounded-lg border-1 border-slate-300 ${EventBackground}` +
          (showConnectingLine ? " mb-3" : "")
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
              <div className="flex-shrink-0 w-6 h-6 flex items-center justify-center">
                {React.createElement(resolveIcon(item.nodeIconSlug || item.nodeIcon), {
                  size: 16,
                  className: "text-gray-600",
                })}
              </div>
            )}
            <span className="text-lg text-zinc-700 font-inter truncate min-w-0 font-semibold">
              {item.nodeDisplayName || item.nodeName || item.componentName}
            </span>
          </div>
          <div
            className={`uppercase text-sm py-[1px] px-[6px] font-semibold rounded flex items-center justify-center text-white ${EventBadgeColor}`}
          >
            <span>{state}</span>
          </div>
        </div>

        {/* Second row: Time ago and duration */}
        <div className="flex items-center mt-0 ml-8 gap-2">
          <span className="text-sm text-gray-500">
            {formatTimeAgo(new Date(item.originalExecution?.createdAt || ""))}
            {item.originalExecution?.state === "STATE_FINISHED" &&
             item.originalExecution?.createdAt &&
             item.originalExecution?.updatedAt && (
              <>
                <span className="mx-1">•</span>
                <span>Duration: {calcRelativeTimeFromDiff(
                  new Date(item.originalExecution.updatedAt).getTime() -
                  new Date(item.originalExecution.createdAt).getTime()
                )}</span>
              </>
            )}
          </span>
        </div>

        {/* Expandable content */}
        {isOpen && item.tabData && (
          <div
            className="mt-3 ml-8 rounded-sm bg-white outline outline-black/15 text-gray-500 w-[calc(100%-2rem)] mb-0.5"
            onClick={(e) => e.stopPropagation()}
          >
            {/* Tab Navigation */}
            <div className="flex justify-between items-center border-b-1 border-gray-200">
              <div className="flex">
                {item.tabData.current && (
                  <button
                    onClick={() => setActiveTab("current")}
                    className={`px-5 py-1 text-sm font-medium rounded-tl-md  ${
                      activeTab === "current"
                        ? "text-black border-b-1 border-black"
                        : "text-gray-500 hover:text-gray-700 hover:bg-gray-50"
                    }`}
                  >
                    Details
                  </button>
                )}
              </div>
              {item.tabData.payload && (
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

            {/* Tab Content */}
            {activeTab === "current" && item.tabData.current && (
              <div className="w-full flex flex-col gap-1 items-center justify-between my-1 px-2 py-2">
                {Object.entries(item.tabData.current).map(([key, value]) => {
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

            {activeTab === "payload" && item.tabData.payload && (
              <div className="w-full px-2 py-2">
                <div className="flex items-center justify-between mb-2 relative">
                  <div className="flex items-center gap-1 absolute right-2 top-4">
                    <SimpleTooltip content={payloadCopied ? "Copied!" : "Copy Link"} hideOnClick={false}>
                      <button
                        onClick={() => copyPayloadToClipboard(item.tabData!.payload)}
                        className="p-1 hover:bg-gray-100 rounded text-gray-500 hover:text-gray-700"
                      >
                        {React.createElement(resolveIcon("copy"), { size: 16 })}
                      </button>
                    </SimpleTooltip>
                    <SimpleTooltip content="Payload">
                      <button
                        onClick={() => {
                          setModalPayload(item.tabData!.payload);
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

      {/* Connecting line */}
      {showConnectingLine && <div className="absolute left-5 -bottom-3 w-[1px] h-3 bg-gray-300 z-10" />}
    </div>
  );
};
