import { Link } from "react-router-dom";
import { Heading } from "../../components/Heading/heading";
import { Text } from "../../components/Text/text";
import { CanvasActionsMenu } from "./CanvasActionsMenu";
import { CanvasMiniMap } from "./CanvasMiniMap";
import type { CanvasCardData, CanvasGroupData } from "./shared";

interface CanvasCardProps {
  canvas: CanvasCardData;
  canvasGroups: CanvasGroupData[];
  organizationId: string;
  onEdit: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

export function CanvasCard({
  canvas,
  canvasGroups,
  organizationId,
  onEdit,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: CanvasCardProps) {
  const canvasHref = `/${organizationId}/canvases/${canvas.id}`;
  const previewNodes = canvas.nodes || [];
  const previewEdges = canvas.edges || [];

  return (
    <div className="relative min-h-40 bg-white dark:bg-gray-800 rounded-md outline outline-gray-950/15 hover:shadow-md transition-shadow cursor-pointer">
      <Link to={canvasHref} aria-label={`Open canvas ${canvas.name}`} className="absolute inset-0 rounded-md" />
      <div className="pointer-events-none relative flex flex-col h-full">
        <div className="p-3">
          <div className="flex items-start justify-between gap-3">
            <div className="flex flex-col flex-1 min-w-0">
              <Heading
                level={3}
                className="mb-0 line-clamp-2 !text-sm font-medium text-gray-800 transition-colors !leading-5"
              >
                <span className="truncate">{canvas.name}</span>
              </Heading>
            </div>
            <div className="pointer-events-auto">
              <CanvasActionsMenu
                canvas={canvas}
                canvasGroups={canvasGroups}
                organizationId={organizationId}
                onEdit={onEdit}
                canUpdateCanvases={canUpdateCanvases}
                canDeleteCanvases={canDeleteCanvases}
                permissionsLoading={permissionsLoading}
              />
            </div>
          </div>

          {canvas.description ? (
            <div className="mb-3">
              <Text className="line-clamp-2 text-left text-[12px] !leading-normal text-gray-800 dark:text-gray-400">
                {canvas.description}
              </Text>
            </div>
          ) : null}

          <div className="flex justify-between items-center">
            <p className="mt-1 text-left text-[11px] leading-none text-gray-500 dark:text-gray-400">
              {canvas.createdBy?.name ? (
                <>
                  Created by {canvas.createdBy.name}, on {canvas.createdAt}
                </>
              ) : (
                <>Created on {canvas.createdAt}</>
              )}
            </p>
          </div>
        </div>

        <CanvasMiniMap nodes={previewNodes} edges={previewEdges} />
      </div>
    </div>
  );
}
