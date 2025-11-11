/* eslint-disable @typescript-eslint/no-explicit-any */
import { resolveIcon } from "@/lib/utils";
import { SidebarEventActionsMenu } from "./SidebarEventActionsMenu";
import React, { useState, useEffect, useCallback, useMemo } from "react";
import { SidebarEvent } from "../types";

export enum ChainExecutionState {
  COMPLETED = "completed",
  FAILED = "failed",
  RUNNING = "running",
}

export interface ExecutionChainItem {
  name: string;
  state: ChainExecutionState;
  children?: Array<{ name: string; state: ChainExecutionState }>;
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
  variant?: "latest" | "queue";
  isOpen: boolean;
  onToggleOpen: (eventId: string) => void;
  onEventClick?: (event: SidebarEvent) => void;
  tabData?: TabData;
  onCancelQueueItem?: (id: string) => void;
  onPassThrough?: (executionId: string) => void;
  supportsPassThrough?: boolean;
}

export const SidebarEventItem: React.FC<SidebarEventItemProps> = ({
  event,
  index,
  variant = "latest",
  isOpen,
  onToggleOpen,
  onEventClick,
  tabData,
  onCancelQueueItem,
  onPassThrough,
  supportsPassThrough,
}) => {
  // Determine default active tab based on available data
  const getDefaultActiveTab = useCallback((): "current" | "root" | "payload" | "executionChain" => {
    if (!tabData) return "current";
    if (tabData.current) return "current";
    if (tabData.root) return "root";
    if (tabData.payload) return "payload";
    if (tabData.executionChain) return "executionChain";
    return "current";
  }, [tabData]);

  const [activeTab, setActiveTab] = useState<"current" | "root" | "payload" | "executionChain">(getDefaultActiveTab());

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
      } else if (activeTab === "executionChain" && !tabData.executionChain) {
        setActiveTab(defaultTab);
      }
    }
  }, [tabData, activeTab, getDefaultActiveTab]);

  let EventIcon = resolveIcon("check");
  let EventColor = "text-green-700";
  let EventBackground = "bg-green-200";
  let iconBorderColor = "border-gray-700";
  let iconSize = 8;
  let iconContainerSize = 4;
  let iconStrokeWidth = 3;
  let animation = "";

  switch (event.state) {
    case "processed":
      EventIcon = resolveIcon("check");
      EventColor = "text-green-700";
      EventBackground = "bg-green-200";
      iconBorderColor = "border-green-700";
      iconSize = 8;
      break;
    case "discarded":
      EventIcon = resolveIcon("x");
      EventColor = "text-red-700";
      EventBackground = "bg-red-200";
      iconBorderColor = "border-red-700";
      iconSize = 8;
      break;
    case "waiting":
      if (variant === "queue") {
        // Match node card styling (neutral grey + dashed icon)
        EventIcon = resolveIcon("circle-dashed");
        EventColor = "text-gray-500";
        EventBackground = "bg-gray-100";
        iconBorderColor = "";
        iconSize = 20;
        iconContainerSize = 5;
        iconStrokeWidth = 2;
        animation = "";
      } else {
        EventIcon = resolveIcon("refresh-cw");
        EventColor = "text-blue-700";
        EventBackground = "bg-blue-100";
        iconBorderColor = "";
        iconSize = 17;
        iconContainerSize = 5;
        iconStrokeWidth = 2;
        animation = "animate-spin";
      }
      break;
    case "running":
      EventIcon = resolveIcon("refresh-cw");
      EventColor = "text-blue-700";
      EventBackground = "bg-blue-100";
      iconBorderColor = "";
      iconSize = 17;
      iconContainerSize = 5;
      iconStrokeWidth = 2;
      animation = "animate-spin";
      break;
  }

  const totalExecutionsCount = useMemo(() => {
    if (!tabData?.executionChain) return 0;

    const childCount = tabData.executionChain.reduce((acc, execution) => acc + (execution.children?.length || 0), 0);
    return childCount + tabData.executionChain.length;
  }, [tabData?.executionChain]);

  return (
    <div
      key={event.title + index}
      className={`flex flex-col items-center justify-between gap-1 px-2 py-1.5 rounded-md ${EventBackground} ${EventColor}`}
    >
      <div className="flex items-center gap-3 rounded-md w-full min-w-0">
        <div
          className="flex items-center gap-2 min-w-0 flex-1 cursor-pointer"
          onClick={(e) => {
            e.stopPropagation();
            onToggleOpen(event.id);
            onEventClick?.(event);
          }}
        >
          <div
            className={`w-${iconContainerSize} h-${iconContainerSize} flex-shrink-0 rounded-full flex items-center justify-center border-[1.5px] ${EventColor} ${iconBorderColor} ${animation}`}
          >
            <EventIcon size={iconSize} strokeWidth={iconStrokeWidth} className="thick" />
          </div>
          <span className="truncate text-sm text-black font-medium">{event.title}</span>
        </div>
        {event.subtitle && (
          <span className="text-sm text-gray-500 truncate flex-shrink-0 max-w-[40%]">{event.subtitle}</span>
        )}
        {/* Actions dropdown */}
        <SidebarEventActionsMenu
          eventId={event.id}
          executionId={(event.executionId || (tabData?.current?.["Execution ID"] as string)) as string | undefined}
          onCancelQueueItem={onCancelQueueItem}
          onPassThrough={onPassThrough}
          supportsPassThrough={supportsPassThrough}
          eventState={event.state}
        />
      </div>
      {isOpen && ((event.values && Object.entries(event.values).length > 0) || tabData) && (
        <div className="rounded-sm bg-white border-1 border-gray-800 text-gray-500 w-full">
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
                {tabData.executionChain && (
                  <button
                    onClick={() => setActiveTab("executionChain")}
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
                  {React.createElement(resolveIcon("code"), { size: 14 })}
                  Payload
                </button>
              )}
            </div>
          )}

          {/* Tab Content */}
          {tabData && activeTab === "current" && tabData.current && (
            <div className="w-full flex flex-col gap-1 items-center justify-between mt-1 px-2 py-2">
              {Object.entries(tabData.current).map(([key, value]) => (
                <div key={key} className="flex items-center gap-1 px-2 rounded-md w-full min-w-0 font-medium">
                  <span className="text-sm flex-shrink-0 text-right w-[30%] truncate" title={key}>
                    {key}:
                  </span>
                  <span
                    className="text-sm flex-1 truncate text-left w-[70%] hover:underline text-gray-800 truncate"
                    title={String(value)}
                  >
                    {String(value)}
                  </span>
                </div>
              ))}
            </div>
          )}

          {tabData && activeTab === "root" && tabData.root && (
            <div className="w-full flex flex-col gap-1 items-center justify-between mt-1 px-2 py-2">
              {Object.entries(tabData.root).map(([key, value]) => (
                <div key={key} className="flex items-center gap-1 px-2 rounded-md w-full min-w-0 font-medium">
                  <span className="text-sm flex-shrink-0 text-right w-[30%] truncate" title={key}>
                    {key}:
                  </span>
                  <span
                    className="text-sm flex-1 truncate text-left w-[70%] hover:underline text-gray-800 truncate"
                    title={String(value)}
                  >
                    {String(value)}
                  </span>
                </div>
              ))}
            </div>
          )}

          {tabData && activeTab === "payload" && tabData.payload && (
            <div className="w-full px-2 py-2">
              <pre className="text-xs bg-gray-50 p-2 rounded border overflow-x-auto">
                {typeof tabData.payload === "string" ? tabData.payload : JSON.stringify(tabData.payload, null, 2)}
              </pre>
            </div>
          )}

          {tabData && activeTab === "executionChain" && tabData.executionChain && (
            <div className="w-full flex flex-col gap-2 px-2 py-2">
              <div className="text-sm text-gray-500 ml-2">
                {totalExecutionsCount} execution{totalExecutionsCount === 1 ? "" : "s"}
              </div>
              {tabData.executionChain.map((execution, index) => (
                <div key={index} className="flex flex-col gap-1">
                  {/* Main execution */}
                  <div className="flex items-center gap-2 px-2 rounded-md w-full min-w-0">
                    <div className="flex-shrink-0">
                      {execution.state === ChainExecutionState.COMPLETED
                        ? React.createElement(resolveIcon("circle-check"), {
                            size: 16,
                            className: "text-green-600",
                          })
                        : execution.state === ChainExecutionState.FAILED
                          ? React.createElement(resolveIcon("x"), {
                              size: 16,
                              className: "text-red-600",
                            })
                          : execution.state === ChainExecutionState.RUNNING
                            ? React.createElement(resolveIcon("refresh-cw"), {
                                size: 16,
                                className: "text-blue-600 animate-spin",
                              })
                            : React.createElement(resolveIcon("circle"), {
                                size: 16,
                                className: "text-gray-400",
                              })}
                    </div>
                    <span className="text-sm text-gray-800 truncate flex-1">{execution.name}</span>
                  </div>
                  {/* Children executions */}
                  {execution.children &&
                    execution.children.map((child, childIndex) => (
                      <div
                        key={`${index}-${childIndex}`}
                        className="flex items-center gap-2 px-2 rounded-md w-full min-w-0"
                      >
                        <div className="flex-shrink-0">
                          {React.createElement(resolveIcon("corner-down-right"), {
                            size: 16,
                            className: "text-gray-400",
                          })}
                        </div>
                        <div className="flex-shrink-0">
                          {child.state === ChainExecutionState.COMPLETED
                            ? React.createElement(resolveIcon("circle-check"), {
                                size: 16,
                                className: "text-green-600",
                              })
                            : child.state === ChainExecutionState.FAILED
                              ? React.createElement(resolveIcon("x"), {
                                  size: 16,
                                  className: "text-red-600",
                                })
                              : child.state === ChainExecutionState.RUNNING
                                ? React.createElement(resolveIcon("refresh-cw"), {
                                    size: 16,
                                    className: "text-blue-600 animate-spin",
                                  })
                                : React.createElement(resolveIcon("circle"), {
                                    size: 16,
                                    className: "text-gray-400",
                                  })}
                        </div>
                        <span className="text-sm text-gray-700 truncate flex-1">{child.name}</span>
                      </div>
                    ))}
                </div>
              ))}
            </div>
          )}

          {/* Fallback to original values display if no tabData */}
          {!tabData && event.values && Object.entries(event.values).length > 0 && (
            <div className="w-full flex flex-col gap-1 items-center justify-between mt-1 px-2 py-2">
              {Object.entries(event.values || {}).map(([key, value]) => (
                <div key={key} className="flex items-center gap-1 px-2 rounded-md w-full min-w-0 font-medium">
                  <span className="text-sm flex-shrink-0 text-right w-[30%] truncate" title={key}>
                    {key}:
                  </span>
                  <span
                    className="text-sm flex-1 truncate text-left w-[70%] hover:underline text-gray-800 truncate"
                    title={value}
                  >
                    {value}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
};
