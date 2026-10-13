import { Heading } from "@/components/Heading/heading";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Link } from "react-router";
import { Star } from "lucide-react";
import type { ReactNode } from "react";
import { appPath } from "@/lib/appPaths";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import { CanvasActionsMenu } from "./CanvasActionsMenu";
import { CanvasCardDescription } from "./CanvasCardDescription";
import type { CanvasCardData } from "./types";

interface CanvasCardsGridProps {
  canvases: CanvasCardData[];
  organizationId: string;
  onEditCanvas: (canvas: CanvasCardData) => void;
  onToggleStar: (canvasId: string, starred: boolean) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

export function CanvasCardsGrid({
  canvases,
  organizationId,
  onEditCanvas,
  onToggleStar,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: CanvasCardsGridProps) {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
      {canvases.map((canvas) => (
        <CanvasCard
          key={canvas.id}
          canvas={canvas}
          organizationId={organizationId}
          onEdit={onEditCanvas}
          onToggleStar={onToggleStar}
          canUpdateCanvases={canUpdateCanvases}
          canDeleteCanvases={canDeleteCanvases}
          permissionsLoading={permissionsLoading}
        />
      ))}
    </div>
  );
}

interface CanvasCardProps {
  canvas: CanvasCardData;
  organizationId: string;
  onEdit: (canvas: CanvasCardData) => void;
  onToggleStar: (canvasId: string, starred: boolean) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

function CanvasCard({
  canvas,
  organizationId,
  onEdit,
  onToggleStar,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: CanvasCardProps) {
  const canvasHref = appPath(organizationId, canvas.id);

  return (
    <div
      className={cn(
        "relative flex h-full flex-col rounded-md bg-white shadow-sm transition-[box-shadow,outline-color] cursor-pointer hover:shadow-md",
        "outline outline-1 -outline-offset-1 outline-slate-950/10 hover:outline-slate-950/15 dark:outline-gray-600/25 dark:hover:outline-gray-600/40",
        appDarkModeClasses.surfaceRaised,
      )}
    >
      <Link to={canvasHref} aria-label={`Open canvas ${canvas.name}`} className="absolute inset-0 rounded-md" />
      <div className="pointer-events-none relative flex flex-1 flex-col">
        <div className="p-3 pb-0">
          <div className="flex items-start justify-between gap-3">
            <div className="flex flex-col flex-1 min-w-0">
              <Heading
                level={3}
                className="mb-0 line-clamp-2 !text-base font-medium text-gray-800 transition-colors !leading-6 dark:text-white"
              >
                <span className="truncate">{canvas.name}</span>
              </Heading>
            </div>
            <div className="pointer-events-auto flex items-center gap-1">
              <CanvasPreferenceButton
                active={Boolean(canvas.isStarred)}
                activeLabel={`Unstar app ${canvas.name}`}
                inactiveLabel={`Star app ${canvas.name}`}
                activeTooltip="Unstar"
                inactiveTooltip="Star"
                onClick={() => onToggleStar(canvas.id, !canvas.isStarred)}
                icon={<Star size={15} className={cn(canvas.isStarred && "fill-current")} aria-hidden />}
              />
              <CanvasActionsMenu
                canvas={canvas}
                organizationId={organizationId}
                onEdit={onEdit}
                canUpdateCanvases={canUpdateCanvases}
                canDeleteCanvases={canDeleteCanvases}
                permissionsLoading={permissionsLoading}
              />
            </div>
          </div>

          {canvas.description ? <CanvasCardDescription description={canvas.description} /> : null}
        </div>

        <div className="mt-auto border-t border-gray-950/10 px-3 pb-3 pt-3 dark:border-gray-700/70">
          <p className="text-left text-[11px] leading-none text-gray-500 dark:text-gray-400">
            Created by {canvas.createdBy.name}, on {canvas.createdAt}
          </p>
        </div>
      </div>
    </div>
  );
}

interface CanvasPreferenceButtonProps {
  active: boolean;
  activeLabel: string;
  inactiveLabel: string;
  activeTooltip: string;
  inactiveTooltip: string;
  icon: ReactNode;
  onClick: () => void;
}

function CanvasPreferenceButton({
  active,
  activeLabel,
  inactiveLabel,
  activeTooltip,
  inactiveTooltip,
  icon,
  onClick,
}: CanvasPreferenceButtonProps) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label={active ? activeLabel : inactiveLabel}
          onClick={(event) => {
            event.preventDefault();
            event.stopPropagation();
            onClick();
          }}
          className={cn(
            "rounded-md text-gray-400 hover:bg-gray-100 hover:text-gray-800 dark:hover:bg-gray-700 dark:hover:text-white",
            active &&
              "bg-blue-50 text-blue-600 hover:bg-blue-100 hover:text-blue-700 dark:bg-blue-950 dark:text-blue-300",
          )}
        >
          {icon}
        </Button>
      </TooltipTrigger>
      <TooltipContent>{active ? activeTooltip : inactiveTooltip}</TooltipContent>
    </Tooltip>
  );
}
