import { calcRelativeTimeFromDiff, resolveIcon } from "@/lib/utils";
import React from "react";
import { ComponentHeader } from "../componentHeader";
import { CollapsedComponent } from "../collapsedComponent";
import { SelectionWrapper } from "../selectionWrapper";
import { ComponentActionsProps } from "../types/componentActions";
import { SpecsTooltip } from "../componentBase/SpecsTooltip";
import { ComponentBaseSpecValue } from "../componentBase";
import Tippy from '@tippyjs/react/headless';
import JsonView from '@uiw/react-json-view';
import { lightTheme } from '@uiw/react-json-view/light';
import 'tippy.js/dist/tippy.css';

export type HttpState = "success" | "failed" | "running";

export interface HttpExecutionItem {
  statusCode?: number;
  receivedAt?: Date;
  state?: HttpState;
}

export interface HttpProps extends ComponentActionsProps {
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;
  headerColor: string;
  title: string;
  method?: string;
  url?: string;
  payload?: Record<string, any>;
  headers?: Array<{ name: string; value: string }>;
  lastExecution?: HttpExecutionItem;
  collapsed?: boolean;
  collapsedBackground?: string;
  selected?: boolean;
  hideLastRun?: boolean;

  onToggleCollapse?: () => void;
}

export const Http: React.FC<HttpProps> = ({
  iconSrc,
  iconSlug = "globe",
  iconColor,
  iconBackground,
  headerColor,
  title,
  method,
  url,
  payload,
  headers,
  lastExecution,
  collapsed = false,
  collapsedBackground,
  selected = false,
  hideLastRun = false,
  onToggleCollapse,
  onRun,
  onEdit,
  onConfigure,
  onDuplicate,
  onDeactivate,
  onToggleView,
  onDelete,
  isCompactView,
}) => {
  const getStateIcon = React.useCallback((state: HttpState) => {
    if (state === "success") return resolveIcon("check");
    if (state === "running") return resolveIcon("refresh-cw");
    return resolveIcon("x");
  }, []);

  const getStateColor = React.useCallback((state: HttpState) => {
    if (state === "success") return "text-green-700";
    if (state === "running") return "text-blue-800";
    return "text-red-700";
  }, []);

  const getStateBackground = React.useCallback((state: HttpState) => {
    if (state === "success") return "bg-green-200";
    if (state === "running") return "bg-sky-100";
    return "bg-red-200";
  }, []);

  const getStateIconBackground = React.useCallback((state: HttpState) => {
    if (state === "success") return "bg-green-600";
    if (state === "running") return "bg-none animate-spin";
    return "bg-red-600";
  }, []);

  const getStateIconColor = React.useCallback((state: HttpState) => {
    if (state === "success") return "text-white";
    if (state === "running") return "text-blue-800";
    return "text-white";
  }, []);

  // Convert headers to spec values for tooltip
  const headerSpecValues: ComponentBaseSpecValue[] = React.useMemo(() => {
    if (!headers || headers.length === 0) return [];
    return headers.map(header => ({
      badges: [
        {
          label: header.name,
          bgColor: "bg-blue-100",
          textColor: "text-blue-800"
        },
        {
          label: header.value,
          bgColor: "bg-gray-100",
          textColor: "text-gray-800"
        }
      ]
    }));
  }, [headers]);

  if (collapsed) {
    return (
      <SelectionWrapper selected={selected}>
        <CollapsedComponent
          iconSrc={iconSrc}
          iconSlug={iconSlug}
          iconColor={iconColor}
          iconBackground={iconBackground}
          title={title}
          collapsedBackground={collapsedBackground}
          shape="rounded"
          onDoubleClick={onToggleCollapse}
          onRun={onRun}
          onEdit={onEdit}
          onConfigure={onConfigure}
          onDuplicate={onDuplicate}
          onDeactivate={onDeactivate}
          onToggleView={onToggleView}
          onDelete={onDelete}
          isCompactView={isCompactView}
        >
          {method && url && (
            <div className="flex items-center gap-2 text-xs text-gray-500 mt-1">
              <span className="font-medium">{method}</span>
              <span className="truncate max-w-[150px]">{url}</span>
            </div>
          )}
        </CollapsedComponent>
      </SelectionWrapper>
    );
  }

  return (
    <SelectionWrapper selected={selected}>
      <div className="flex flex-col border-2 border-border rounded-md w-[26rem] bg-white">
        <ComponentHeader
          iconSrc={iconSrc}
          iconSlug={iconSlug}
          iconBackground={iconBackground}
          iconColor={iconColor}
          headerColor={headerColor}
          title={title}
          onDoubleClick={onToggleCollapse}
          onRun={onRun}
          onEdit={onEdit}
          onConfigure={onConfigure}
          onDuplicate={onDuplicate}
          onDeactivate={onDeactivate}
          onToggleView={onToggleView}
          onDelete={onDelete}
          isCompactView={isCompactView}
        />

        {/* Request Details Section */}
        {(method || url) && (
          <div className="px-4 py-3 border-b">
            <div className="flex flex-col gap-2 text-sm">
              <div className="flex items-center gap-2">
                <span className="px-2 py-1 rounded-md text-xs font-mono font-medium bg-blue-100 text-blue-800">
                  {method || "GET"}
                </span>
                <span className="text-gray-700 font-mono text-xs truncate">
                  {url}
                </span>
              </div>

              {/* Payload Tooltip */}
              {payload && Object.keys(payload).length > 0 && (
                <div className="flex items-center gap-2">
                  <Tippy
                    render={() => (
                      <div className="bg-white border-2 border-gray-200 rounded-md max-w-[500px] max-h-[400px] overflow-auto text-left">
                        <div className="p-2">
                          <JsonView
                            value={payload}
                            style={{
                              fontSize: '12px',
                              fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Consolas, "Liberation Mono", Menlo, monospace',
                              backgroundColor: 'transparent',
                              textAlign: 'left',
                              ...lightTheme
                            }}
                            displayDataTypes={false}
                            displayObjectSize={false}
                            enableClipboard={false}
                            collapsed={1}
                          />
                        </div>
                      </div>
                    )}
                    placement="top"
                    interactive={true}
                    delay={200}
                  >
                    <span className="text-xs bg-gray-500 px-2 py-1 rounded-md text-white font-mono font-medium cursor-help">
                      Payload
                    </span>
                  </Tippy>
                </div>
              )}

              {/* Headers Tooltip */}
              {headerSpecValues.length > 0 && (
                <div className="flex items-center gap-2">
                  <SpecsTooltip
                    specTitle="headers"
                    tooltipTitle="request headers"
                    specValues={headerSpecValues}
                  >
                    <span className="text-xs bg-gray-500 px-2 py-1 rounded-md text-white font-mono font-medium cursor-help">
                      {headerSpecValues.length} header{headerSpecValues.length > 1 ? "s" : ""}
                    </span>
                  </SpecsTooltip>
                </div>
              )}

            </div>
          </div>
        )}

        {/* Last Run Section */}
        {!hideLastRun && (
          <div className="px-4 py-3 border-b">
            <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
              <span className="uppercase text-sm font-medium">Last Run</span>
            </div>

            {lastExecution && lastExecution.state && lastExecution.receivedAt ? (
              <div className="flex flex-col gap-2">
                <div
                  className={`flex items-center justify-between gap-3 px-2 py-2 rounded-md ${getStateBackground(lastExecution.state)} ${getStateColor(lastExecution.state)}`}
                >
                  <div className="flex items-center gap-2 min-w-0 flex-1">
                    <div
                      className={`w-5 h-5 flex-shrink-0 rounded-full flex items-center justify-center ${getStateIconBackground(lastExecution.state)}`}
                    >
                      {React.createElement(getStateIcon(lastExecution.state), {
                        size: lastExecution.state === "running" ? 16 : 12,
                        className: getStateIconColor(lastExecution.state),
                      })}
                    </div>
                    {lastExecution.statusCode ? (
                      <span className="text-sm font-medium">
                        Status: {lastExecution.statusCode}
                      </span>
                    ) : (
                      <span className="text-sm">
                        {lastExecution.state === "running" ? "Running..." : "Failed"}
                      </span>
                    )}
                  </div>
                  <span className="text-xs text-gray-500">
                    {calcRelativeTimeFromDiff(
                      new Date().getTime() - lastExecution.receivedAt.getTime()
                    )}
                  </span>
                </div>
              </div>
            ) : (
              <div className="flex items-center gap-3 px-2 py-2 rounded-md bg-gray-100 text-gray-500">
                <div className="w-5 h-5 rounded-full flex items-center justify-center bg-gray-400">
                  <div className="w-2 h-2 rounded-full bg-white"></div>
                </div>
                <span className="text-sm">No executions received yet</span>
              </div>
            )}
          </div>
        )}
      </div>
    </SelectionWrapper>
  );
};
