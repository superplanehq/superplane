import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Link } from "react-router-dom";
import { AppDotGrid } from "./AppDotGrid";
import { CanvasActionsMenu } from "./CanvasActionsMenu";
import type { CanvasCardData, CanvasFolderData } from "./types";

interface CanvasCardsGridProps {
  canvases: CanvasCardData[];
  canvasFolders: CanvasFolderData[];
  organizationId: string;
  onEditCanvas: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

export function CanvasCardsGrid({
  canvases,
  canvasFolders,
  organizationId,
  onEditCanvas,
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
          canvasFolders={canvasFolders}
          organizationId={organizationId}
          onEdit={onEditCanvas}
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
  canvasFolders: CanvasFolderData[];
  organizationId: string;
  onEdit: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

function CanvasCard({
  canvas,
  canvasFolders,
  organizationId,
  onEdit,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: CanvasCardProps) {
  const canvasHref = `/${organizationId}/canvases/${canvas.id}`;

  return (
    <div className="relative flex min-h-48 flex-col overflow-hidden rounded-md bg-white outline outline-gray-950/15 transition-shadow hover:shadow-md dark:bg-gray-800 cursor-pointer">
      <Link to={canvasHref} aria-label={`Open canvas ${canvas.name}`} className="absolute inset-0 rounded-md" />
      <div className="pointer-events-none relative flex flex-1">
        <div className="flex shrink-0 items-start border-r border-slate-950/10 p-3 dark:border-slate-50/10">
          <AppDotGrid seed={canvas.id} />
        </div>

        <div className="flex min-w-0 flex-1 flex-col p-3">
          <div className="flex items-start justify-between gap-3">
            <Heading
              level={3}
              className="mb-0 line-clamp-2 !text-lg font-medium text-gray-800 transition-colors !leading-6"
            >
              {canvas.name}
            </Heading>
            <div className="pointer-events-auto">
              <CanvasActionsMenu
                canvas={canvas}
                canvasFolders={canvasFolders}
                organizationId={organizationId}
                onEdit={onEdit}
                canUpdateCanvases={canUpdateCanvases}
                canDeleteCanvases={canDeleteCanvases}
                permissionsLoading={permissionsLoading}
              />
            </div>
          </div>

          {canvas.description ? (
            <div className="mt-1.5">
              <Text className="text-left text-[12px] !leading-normal text-gray-800 dark:text-gray-400">
                {canvas.description}
              </Text>
            </div>
          ) : null}

          <p className="mt-auto pt-3 text-left text-[11px] leading-none text-gray-500 dark:text-gray-400">
            Created by {canvas.createdBy.name}, on {canvas.createdAt}
          </p>
        </div>
      </div>
    </div>
  );
}
