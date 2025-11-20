import { calcRelativeTimeFromDiff, resolveIcon } from "@/lib/utils";
import React from "react";
import { ComponentHeader } from "../componentHeader";
import { CollapsedComponent } from "../collapsedComponent";
import { MetadataList, type MetadataItem } from "../metadataList";
import { SelectionWrapper } from "../selectionWrapper";
import { ComponentActionsProps } from "../types/componentActions";
import { SpecsTooltip } from "../componentBase/SpecsTooltip";
import { ComponentBaseSpecValue } from "../componentBase";

export type SemaphoreState = "success" | "failed" | "running";

export interface SemaphoreExecutionItem {
  title: string;
  receivedAt?: Date;
  completedAt?: Date;
  state?: SemaphoreState;
  values?: Record<string, string>;
  duration?: number; // Duration in milliseconds (for finished executions)
}

export interface SemaphoreParameter {
  name: string;
  value: string;
}

export interface SemaphoreProps extends ComponentActionsProps {
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;
  headerColor: string;
  title: string;
  integration?: string;
  metadata: MetadataItem[];
  parameters?: SemaphoreParameter[];
  lastExecution?: SemaphoreExecutionItem;
  collapsed?: boolean;
  collapsedBackground?: string;
  selected?: boolean;
  hideLastRun?: boolean;

  onToggleCollapse?: () => void;
}

export const Semaphore: React.FC<SemaphoreProps> = ({
  iconSrc,
  iconSlug = "workflow",
  iconColor,
  iconBackground,
  headerColor,
  title,
  metadata,
  parameters,
  lastExecution,
  collapsed = false,
  collapsedBackground,
  selected = false,
  hideLastRun = false,
  onToggleCollapse,
  onRun,
  runDisabled,
  runDisabledTooltip,
  onEdit,
  onConfigure,
  onDuplicate,
  onDeactivate,
  onToggleView,
  onDelete,
  isCompactView,
}) => {
  const getStateIcon = React.useCallback((state: SemaphoreState) => {
    if (state === "success") return resolveIcon("check");
    if (state === "running") return resolveIcon("refresh-cw");
    return resolveIcon("x");
  }, []);

  const getStateColor = React.useCallback((state: SemaphoreState) => {
    if (state === "success") return "text-green-700";
    if (state === "running") return "text-blue-800";
    return "text-red-700";
  }, []);

  const getStateBackground = React.useCallback((state: SemaphoreState) => {
    if (state === "success") return "bg-green-200";
    if (state === "running") return "bg-sky-100";
    return "bg-red-200";
  }, []);

  const getStateIconBackground = React.useCallback((state: SemaphoreState) => {
    if (state === "success") return "bg-green-600";
    if (state === "running") return "bg-none animate-spin";
    return "bg-red-600";
  }, []);

  const getStateIconColor = React.useCallback((state: SemaphoreState) => {
    if (state === "success") return "text-white";
    if (state === "running") return "text-blue-800";
    return "text-white";
  }, []);

  // Convert parameters to spec values for tooltip
  const parameterSpecValues: ComponentBaseSpecValue[] = React.useMemo(() => {
    if (!parameters || parameters.length === 0) return [];
    return parameters.map((param) => ({
      badges: [
        {
          label: param.name,
          bgColor: "bg-purple-100",
          textColor: "text-purple-800",
        },
        {
          label: param.value,
          bgColor: "bg-gray-100",
          textColor: "text-gray-800",
        },
      ],
    }));
  }, [parameters]);

  // Live timer for running executions
  const [liveDuration, setLiveDuration] = React.useState<number | null>(null);

  React.useEffect(() => {
    if (lastExecution?.state === "running" && lastExecution.receivedAt) {
      const receivedAt = lastExecution.receivedAt;

      // Calculate initial duration
      setLiveDuration(Date.now() - receivedAt.getTime());

      // Update every second
      const interval = setInterval(() => {
        setLiveDuration(Date.now() - receivedAt.getTime());
      }, 1000);

      return () => clearInterval(interval);
    } else {
      setLiveDuration(null);
    }
  }, [lastExecution?.state, lastExecution?.receivedAt]);

  // Calculate display duration
  const displayDuration = React.useMemo(() => {
    if (lastExecution?.state === "running" && liveDuration !== null) {
      return liveDuration;
    }
    return lastExecution?.duration;
  }, [lastExecution?.state, lastExecution?.duration, liveDuration]);

  // Format timestamp for "Done at"
  const formatTimestamp = (date: Date): string => {
    return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  };

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
          runDisabled={runDisabled}
          runDisabledTooltip={runDisabledTooltip}
          onEdit={onEdit}
          onConfigure={onConfigure}
          onDuplicate={onDuplicate}
          onDeactivate={onDeactivate}
          onToggleView={onToggleView}
          onDelete={onDelete}
          isCompactView={isCompactView}
        >
          <div className="flex flex-col items-center gap-1">
            <MetadataList items={metadata} className="flex flex-col gap-1 text-gray-500" iconSize={12} />
          </div>
        </CollapsedComponent>
      </SelectionWrapper>
    );
  }

  return (
    <SelectionWrapper selected={selected}>
      <div className="flex flex-col outline outline-black-15 rounded-md w-[23rem] bg-white">
        <ComponentHeader
          iconSrc={iconSrc}
          iconSlug={iconSlug}
          iconBackground={iconBackground}
          iconColor={iconColor}
          headerColor={headerColor}
          title={title}
          onDoubleClick={onToggleCollapse}
          onRun={onRun}
          runDisabled={runDisabled}
          runDisabledTooltip={runDisabledTooltip}
          onEdit={onEdit}
          onConfigure={onConfigure}
          onDuplicate={onDuplicate}
          onDeactivate={onDeactivate}
          onToggleView={onToggleView}
          onDelete={onDelete}
          isCompactView={isCompactView}
        />

        {/* Metadata Section */}
        <MetadataList items={metadata} />

        {/* Parameters Section */}
        {parameterSpecValues.length > 0 && (
          <div className="px-4 py-3 border-b">
            <div className="flex items-start gap-2">
              <SpecsTooltip specTitle="parameters" tooltipTitle="workflow parameters" specValues={parameterSpecValues}>
                <span className="text-xs bg-gray-200 px-2 py-1 rounded-md text-gray-800 font-mono font-medium cursor-help">
                  {parameterSpecValues.length} parameter{parameterSpecValues.length > 1 ? "s" : ""}
                </span>
              </SpecsTooltip>
            </div>
          </div>
        )}

        {/* Last Run Section */}
        {!hideLastRun && (
          <div className="px-4 py-3 border-b">
            <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
              <span className="uppercase text-xs font-semibold tracking-wide">Last Run</span>
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
                    <span className="text-sm font-medium truncate">{lastExecution.title}</span>
                  </div>
                  <span className="text-xs text-gray-500">
                    {lastExecution.state === "running" && displayDuration !== undefined && displayDuration !== null
                      ? `Running for: ${calcRelativeTimeFromDiff(displayDuration)}`
                      : lastExecution.completedAt
                        ? `Done at: ${formatTimestamp(lastExecution.completedAt)}`
                        : ""}
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
