/* eslint-disable @typescript-eslint/no-explicit-any */
import { Button } from "@/components/ui/button";
import { resolveIcon, isUrl } from "@/lib/utils";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import type { SidebarEvent } from "../types";
import { SidebarEventActionsMenu } from "./SidebarEventActionsMenu";
import JsonView from "@uiw/react-json-view";
import { SimpleTooltip } from "../SimpleTooltip";
import type { EventState, EventStateMap, EventStateStyle } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import type { CanvasesCanvasNodeExecution } from "@/api-client";

export interface TabData {
  current?: Record<string, any>;
  root?: Record<string, any>;
  payload?: any;
}

interface SidebarEventItemProps {
  event: SidebarEvent;
  index: number;
  totalItems?: number;
  variant?: "latest" | "queue";
  isOpen: boolean;
  onToggleOpen: (eventId: string) => void;
  onEventClick?: (event: SidebarEvent) => void;
  tabData?: TabData;
  onCancelQueueItem?: (id: string) => void;
  onCancelExecution?: (executionId: string) => void;
  onReEmit?: (nodeId: string, eventOrExecutionId: string) => void;
  getExecutionState?: (
    nodeId: string,
    execution: CanvasesCanvasNodeExecution,
  ) => { map: EventStateMap; state: EventState };
}

export const SidebarEventItem: React.FC<SidebarEventItemProps> = ({
  event,
  index,
  totalItems,
  isOpen,
  onToggleOpen,
  onEventClick,
  tabData,
  onCancelQueueItem,
  onCancelExecution,
  onReEmit,
  getExecutionState,
}) => {
  // Determine default active tab based on available data
  const getDefaultActiveTab = useCallback((): "current" | "root" | "payload" => {
    if (!tabData) return "current";
    if (tabData.current) return "current";
    if (tabData.root) return "root";
    if (tabData.payload) return "payload";
    return "current";
  }, [tabData]);

  const [activeTab, setActiveTab] = useState<"current" | "root" | "payload">(getDefaultActiveTab());
  const [isPayloadModalOpen, setIsPayloadModalOpen] = useState(false);
  const [modalPayload, setModalPayload] = useState<any>(null);
  const [payloadCopied, setPayloadCopied] = useState(false);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [isHovered, setIsHovered] = useState(false);

  const eventStateStyle: EventStateStyle = useMemo(() => {
    if (!getExecutionState) return DEFAULT_EVENT_STATE_MAP["neutral"];

    if (event.kind === "queue") return DEFAULT_EVENT_STATE_MAP["queued"];

    if (event.kind === "trigger") {
      return DEFAULT_EVENT_STATE_MAP[event.state as EventState] || DEFAULT_EVENT_STATE_MAP["neutral"];
    }

    const { map, state } = getExecutionState(
      event.nodeId || "",
      event.originalExecution as CanvasesCanvasNodeExecution,
    );
    return map[state];
  }, [event.nodeId, event.originalExecution, getExecutionState, event.kind, event.state]);

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
    }
  }, [tabData, activeTab, getDefaultActiveTab]);

  const EventBackground = eventStateStyle.backgroundColor;
  const EventBadgeColor = eventStateStyle.badgeColor;

  // Determine if actions menu should be shown (same logic as in SidebarEventActionsMenu)
  const isWaiting = event.state === "waiting";
  const isQueued = event.state === "queued";
  const isRunning = event.state === "running";

  const showCancel = (event.kind === "queue" && isQueued) || (event.kind === "execution" && (isRunning || isWaiting));
  const showReEmit = event.kind === "trigger";
  const showActionsMenu = showCancel || showReEmit;

  return (
    <div
      key={event.title + index}
      className={
        `cursor-pointer p-2 relative rounded-md border-1 border-edge-strong hover:translate-x-1 transition-transform duration-200 ${EventBackground}` +
        (totalItems && index < totalItems - 1 ? " mb-4" : "")
      }
      data-testid="sidebar-event-item"
      data-event-state={event.state || "unknown"}
      data-event-kind={event.kind || "execution"}
      onClick={(e) => {
        e.stopPropagation();
        onToggleOpen(event.id);
        onEventClick?.(event);
      }}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      {/* First row: Badge and subtitle */}
      <div className="flex items-center justify-between gap-2 min-w-0 flex-1">
        <div
          className={`uppercase text-[11px] py-[1.5px] px-[5px] font-semibold rounded flex items-center tracking-wide justify-center text-white ${EventBadgeColor}`}
        >
          <span>{eventStateStyle.label || event.state || "neutral"}</span>
        </div>
        {event.subtitle && (
          <span className="text-[13px] font-medium truncate flex-shrink-0 max-w-[65%] text-content-primary/50">
            {event.subtitle}
          </span>
        )}
      </div>

      {/* Second row: Event ID and title with actions */}
      <div className="flex items-center mt-1 gap-2">
        <div className="flex items-center gap-2 min-w-0 flex-1 cursor-pointer">
          {event.triggerEventId && (
            <span className="text-[13px] text-content-primary/50 font-mono">#{event.triggerEventId.slice(0, 4)}</span>
          )}
          <span className="text-sm text-content-primary font-inter truncate text-md min-w-0 font-medium">
            {event.title}
          </span>
        </div>
      </div>

      {/* Hover overlay with dropdown menu */}
      {showActionsMenu && (isHovered || isDropdownOpen) && (
        <div className="absolute top-0 right-0 h-full flex items-center bg-transparent">
          <div
            className="h-full bg-surface-raised/50 backdrop-blur-[3px] rounded-r-md shadow-sm p-1 pt-2"
            onClick={(e) => e.stopPropagation()}
          >
            <SidebarEventActionsMenu
              eventId={event.id}
              executionId={event.executionId}
              onCancelQueueItem={onCancelQueueItem}
              onCancelExecution={onCancelExecution}
              eventState={event.state}
              kind={event.kind || "execution"}
              onReEmit={() => {
                if (["queue", "execution"].includes(event.kind || "")) return;
                onReEmit?.(event.nodeId || "", event.id);
              }}
              onOpenChange={setIsDropdownOpen}
            />
          </div>
        </div>
      )}

      {isOpen && ((event.values && Object.entries(event.values).length > 0) || tabData) && (
        <div
          className="mt-3 rounded-sm bg-surface-raised outline outline-edge-default text-content-secondary w-full mb-0.5"
          onClick={(e) => e.stopPropagation()}
        >
          {/* Tab Navigation */}
          {tabData && (
            <div className="flex justify-between items-center border-b-1 border-edge-subtle">
              <div className="flex">
                {tabData.current && (
                  <button
                    onClick={() => setActiveTab("current")}
                    className={`px-5 py-1 text-sm font-medium rounded-tl-md  ${
                      activeTab === "current"
                        ? "border-b-1 border-action-primary text-content-primary"
                        : "text-content-secondary hover:bg-action-neutral-hover hover:text-content-primary"
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
                        ? "border-b-1 border-action-primary text-content-primary"
                        : "text-content-secondary hover:bg-action-neutral-hover hover:text-content-primary"
                    }`}
                  >
                    Root
                  </button>
                )}
              </div>
              {tabData.payload && (
                <button
                  onClick={() => setActiveTab("payload")}
                  className={`px-3 py-1 text-sm font-medium rounded-tr-md flex items-center gap-1 ${
                    activeTab === "payload"
                      ? "border-b-1 border-action-primary bg-surface-subtle text-content-primary"
                      : "border-l-1 border-edge-subtle text-content-secondary hover:bg-action-neutral-hover hover:text-content-primary"
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
            <div className="w-full flex flex-col gap-1 items-center justify-between my-1 px-2 pt-2 pb-3">
              {Object.entries(tabData.current).map(([key, value]) => {
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
                        className="text-sm flex-1 text-left w-[70%] text-content-primary cursor-pointer inline-block overflow-hidden text-ellipsis whitespace-nowrap max-w-full"
                        style={{ textDecoration: "underline", textDecorationThickness: "1px" }}
                        title={stringValue}
                        onClick={(e) => e.stopPropagation()}
                      >
                        {stringValue}
                      </a>
                    ) : (
                      <span
                        className="w-[70%] flex-1 truncate text-left text-[13px] text-content-primary hover:underline"
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
            <div className="w-full flex flex-col gap-1 items-center justify-between my-1 px-2 pt-2 pb-3">
              {Object.entries(tabData.root).map(([key, value]) => {
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
                        className="text-sm flex-1 text-left w-[70%] text-content-primary cursor-pointer inline-block overflow-hidden text-ellipsis whitespace-nowrap max-w-full"
                        style={{ textDecoration: "underline", textDecorationThickness: "1px" }}
                        title={stringValue}
                        onClick={(e) => e.stopPropagation()}
                      >
                        {stringValue}
                      </a>
                    ) : (
                      <span
                        className="w-[70%] flex-1 truncate text-left text-[13px] text-content-primary hover:underline"
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
                      className="p-1 text-content-secondary hover:text-content-primary"
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
                      className="p-1 text-content-secondary hover:text-content-primary"
                    >
                      {React.createElement(resolveIcon("maximize-2"), { size: 16 })}
                    </button>
                  </SimpleTooltip>
                </div>
              </div>
              <div className="h-50 overflow-auto rounded -mt-2">
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

          {/* Fallback to original values display if no tabData */}
          {!tabData && event.values && Object.entries(event.values).length > 0 && (
            <div className="w-full flex flex-col gap-1 items-center justify-between my-1 px-2 pt-2 pb-3">
              {Object.entries(event.values || {}).map(([key, value]) => {
                const isUrlValue = isUrl(value);

                return (
                  <div key={key} className="flex items-center gap-1 px-2 rounded-md w-full min-w-0">
                    <span className="text-[13px] flex-shrink-0 text-right w-[30%] truncate" title={key}>
                      {key}:
                    </span>
                    {isUrlValue ? (
                      <a
                        href={value}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-sm flex-1 text-left w-[70%] text-content-primary cursor-pointer inline-block overflow-hidden text-ellipsis whitespace-nowrap max-w-full"
                        style={{ textDecoration: "underline", textDecorationThickness: "1px" }}
                        title={value}
                        onClick={(e) => e.stopPropagation()}
                      >
                        {value}
                      </a>
                    ) : (
                      <span
                        className="w-[70%] flex-1 truncate text-left text-[13px] text-content-primary hover:underline"
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
        <div className="fixed inset-0 bg-overlay-scrim z-50 flex items-center justify-center p-4">
          <div className="bg-surface-raised rounded-lg w-full max-w-4xl max-h-[90vh] flex flex-col">
            <div className="flex items-center justify-between p-4 border-b border-edge-subtle">
              <h3 className="text-lg font-semibold text-content-primary">Payload</h3>
              <div className="flex items-center gap-2">
                <SimpleTooltip content={payloadCopied ? "Copied!" : "Copy Link"} hideOnClick={false}>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="gap-1"
                    onClick={() => copyPayloadToClipboard(modalPayload)}
                  >
                    {React.createElement(resolveIcon("copy"), { size: 14 })}
                    Copy
                  </Button>
                </SimpleTooltip>
                <button
                  onClick={() => {
                    setIsPayloadModalOpen(false);
                    setModalPayload(null);
                  }}
                  className="p-1 hover:text-content-secondary"
                >
                  {React.createElement(resolveIcon("x"), { size: 16 })}
                </button>
              </div>
            </div>
            <div className="flex-1 overflow-auto bg-surface-raised rounded-b-lg">
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
