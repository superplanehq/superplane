import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import type { CanvasesCanvas, SuperplaneComponentsEdge, SuperplaneComponentsNode } from "@/api-client";
import { Link } from "react-router-dom";
import { Rainbow } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { getIntegrationIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { extractIntegrations, getTemplateTags, countNodesByType } from "./templateMetadata";
import { NodeCountLabel, TagBadges } from "./components/TemplateCardMeta";

interface TemplateCardProps {
  template: CanvasesCanvas;
  organizationId: string;
  showTags?: boolean;
}

function IntegrationIcons({ integrations }: { integrations: string[] }) {
  if (integrations.length === 0) {
    return <span className="text-[11px] text-gray-400 dark:text-gray-500">No integrations needed</span>;
  }

  return (
    <div className="flex items-center gap-1.5 shrink-0">
      {integrations.map((name) => {
        const iconSrc = getIntegrationIconSrc(name);
        if (!iconSrc) return null;
        return (
          <Tooltip key={name}>
            <TooltipTrigger asChild>
              <span className="inline-block h-4 w-4 shrink-0">
                <img src={iconSrc} alt={name} className="h-full w-full object-contain" />
              </span>
            </TooltipTrigger>
            <TooltipContent side="bottom">
              <span className="capitalize">{name}</span>
            </TooltipContent>
          </Tooltip>
        );
      })}
    </div>
  );
}

export function TemplateCard({ template, organizationId, showTags = false }: TemplateCardProps) {
  const metadata = template.metadata;
  const nodes = template.spec?.nodes;
  const previewNodes = (nodes ?? []) as SuperplaneComponentsNode[];
  const previewEdges = (template.spec?.edges ?? []) as SuperplaneComponentsEdge[];
  const templateId = metadata?.id;

  if (!templateId) return null;

  const templateName = metadata?.name ?? "Untitled template";
  const description = metadata?.description ?? "";
  const tags = showTags ? getTemplateTags(metadata?.name) : [];
  const integrations = extractIntegrations(nodes);
  const { components, triggers } = countNodesByType(nodes);

  return (
    <Link
      to={`/${organizationId}/templates/${templateId}`}
      className="min-h-48 bg-white dark:bg-gray-800 rounded-md outline outline-slate-950/10 hover:shadow-md transition-shadow cursor-pointer group flex flex-col"
    >
      <div className="relative">
        <CanvasMiniMap nodes={previewNodes} edges={previewEdges} />
        <div
          className="absolute inset-0 flex items-center justify-center bg-white/80 rounded-t-md opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none"
          aria-hidden
        >
          <span className="text-sm text-gray-800 dark:text-gray-900 bg-white/80 rounded-sm outline outline-1 outline-gray-400 dark:outline-gray-600 px-2 py-1">
            Preview
          </span>
        </div>
      </div>

      <div className="p-4 border-t border-gray-200 dark:border-gray-700 flex flex-col flex-1">
        <Heading
          level={3}
          className="!text-base font-medium text-gray-800 transition-colors mb-1 !leading-6 line-clamp-2"
        >
          {templateName}
        </Heading>

        {description ? (
          <Text className="text-[13px] !leading-normal text-left text-gray-800 dark:text-gray-400 line-clamp-3">
            {description}
          </Text>
        ) : null}

        <NodeCountLabel components={components} triggers={triggers} />

        <div className="mt-auto pt-3 flex items-end justify-between gap-2">
          <TagBadges tags={tags} />
          <IntegrationIcons integrations={integrations} />
        </div>
      </div>
    </Link>
  );
}

interface CanvasMiniMapProps {
  nodes?: SuperplaneComponentsNode[];
  edges?: SuperplaneComponentsEdge[];
}

function CanvasMiniMap({ nodes = [], edges = [] }: CanvasMiniMapProps) {
  const positionedNodes = nodes.filter(
    (node) => typeof node.position?.x === "number" && typeof node.position?.y === "number",
  ) as Array<SuperplaneComponentsNode & { position: { x: number; y: number } }>;

  if (!positionedNodes.length) {
    return (
      <div className="p-4">
        <div className="h-28 w-full bg-transparent flex flex-col items-center justify-center pt-4 gap-1 text-[13px] text-gray-500">
          <Rainbow size={24} className="text-gray-500" />
          Canvas is empty
        </div>
      </div>
    );
  }

  const xs = positionedNodes.map((node) => node.position.x);
  const ys = positionedNodes.map((node) => node.position.y);
  const minX = Math.min(...xs);
  const maxX = Math.max(...xs);
  const minY = Math.min(...ys);
  const maxY = Math.max(...ys);
  const padding = 80;
  const width = Math.max(maxX - minX, 200) + padding * 2;
  const height = Math.max(maxY - minY, 200) + padding * 2;
  const viewBox = `${minX - padding} ${minY - padding} ${width} ${height}`;
  const nodeWidth = Math.min(Math.max(width * 0.08, 30), 80);
  const nodeHeight = nodeWidth * 0.45;

  const nodePositions = new Map<string, { x: number; y: number }>();
  positionedNodes.forEach((node) => {
    const id = node.id || node.name;
    if (!id) return;
    nodePositions.set(id, { x: node.position.x, y: node.position.y });
  });

  const drawableEdges =
    edges?.filter(
      (edge) => edge.sourceId && edge.targetId && nodePositions.has(edge.sourceId) && nodePositions.has(edge.targetId),
    ) || [];

  return (
    <div className="p-4 w-full overflow-hidden">
      <svg
        viewBox={viewBox}
        preserveAspectRatio="xMidYMid meet"
        className="w-full h-28 text-gray-500 dark:text-gray-400"
      >
        {drawableEdges.map((edge) => {
          const source = nodePositions.get(edge.sourceId!);
          const target = nodePositions.get(edge.targetId!);
          if (!source || !target) return null;
          return (
            <line
              key={`${edge.sourceId}-${edge.targetId}`}
              x1={source.x}
              y1={source.y}
              x2={target.x}
              y2={target.y}
              stroke="currentColor"
              strokeWidth={6}
              strokeLinecap="round"
              opacity={0.25}
            />
          );
        })}
        {positionedNodes.map((node) => {
          const id = node.id || node.name || `${node.position.x}-${node.position.y}`;
          return (
            <rect
              key={id}
              x={node.position.x - nodeWidth / 2}
              y={node.position.y - nodeHeight / 2}
              width={nodeWidth}
              height={nodeHeight}
              rx={8}
              ry={8}
              fill="#1f2937"
              opacity={1}
            />
          );
        })}
      </svg>
    </div>
  );
}
