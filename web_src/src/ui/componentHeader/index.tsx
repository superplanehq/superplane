import { resolveIcon } from "@/lib/utils";
import React from "react";
import { toTestId } from "../../lib/testID";
import type { ComponentActionsProps } from "../types/componentActions";

export interface ComponentHeaderProps extends ComponentActionsProps {
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  title: string;
  onDoubleClick?: () => void;
  statusBadgeColor?: string;
}

export const ComponentHeader: React.FC<ComponentHeaderProps> = ({
  iconSrc,
  iconSlug,
  iconColor,
  title,
  onDoubleClick,
  statusBadgeColor,
  isCompactView = false,
  onOpenRunnerLiveLogs,
}) => {
  const Icon = React.useMemo(() => {
    return resolveIcon(iconSlug);
  }, [iconSlug]);

  return (
    <div
      data-testid={toTestId(`node-${title}-header`)}
      data-view-mode={isCompactView ? "compact" : "expanded"}
      className={
        "canvas-node-drag-handle text-left text-lg w-full px-2 py-1.5 flex items-center flex-col rounded-t-md items-center relative" +
        (isCompactView ? "" : " border-b border-slate-950/20")
      }
      onDoubleClick={onDoubleClick}
    >
      <div className="flex w-full min-w-0 items-center gap-2">
        <div className="flex min-w-0 flex-1 items-center">
          <div className="mr-2 flex h-4 w-4 shrink-0 items-center justify-center overflow-hidden">
            {iconSrc ? (
              <img src={iconSrc} alt={title} className="h-4 w-4 shrink-0 object-contain" />
            ) : (
              <Icon size={16} className={iconColor} />
            )}
          </div>
          <h2 className="min-w-0 truncate text-sm font-semibold">{title}</h2>
        </div>
        <div className="flex shrink-0 items-center gap-1.5">
          {onOpenRunnerLiveLogs ? (
            <button
              type="button"
              data-testid="runner-live-logs"
              aria-label="Open logs"
              className="nodrag rounded-md border border-slate-200/90 bg-white/80 px-2 py-0.5 text-xs font-medium text-slate-600 shadow-none transition hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900"
              onClick={(event) => {
                event.preventDefault();
                event.stopPropagation();
                onOpenRunnerLiveLogs();
              }}
            >
              Logs
            </button>
          ) : null}
          {isCompactView && statusBadgeColor ? (
            <span className={`h-2.5 w-2.5 shrink-0 rounded-full ${statusBadgeColor}`} aria-hidden="true" />
          ) : null}
        </div>
      </div>
    </div>
  );
};
