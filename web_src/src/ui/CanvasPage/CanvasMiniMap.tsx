import { MiniMap, type Edge as ReactFlowEdge, type MiniMapNodeProps, type Node as ReactFlowNode } from "@xyflow/react";
import { useCallback, useMemo } from "react";

const MINIMAP_WIDTH = 198;
const MINIMAP_HEIGHT = 150;

type CanvasMiniMapProps = {
  nodes: ReactFlowNode[];
  edges: ReactFlowEdge[];
  isVisible: boolean;
};

export function CanvasMiniMap({ nodes, edges, isVisible }: CanvasMiniMapProps) {
  const minimapNodeCenters = useMemo(() => {
    const centers = new Map<string, { x: number; y: number }>();

    nodes.forEach((node) => {
      const absolutePosition = (node as ReactFlowNode & { positionAbsolute?: { x: number; y: number } })
        .positionAbsolute;
      const x = absolutePosition?.x ?? node.position?.x ?? 0;
      const y = absolutePosition?.y ?? node.position?.y ?? 0;
      const width = node.measured?.width ?? node.width ?? 160;
      const height = node.measured?.height ?? node.height ?? 56;

      centers.set(node.id, {
        x: x + width / 2,
        y: y + height / 2,
      });
    });

    return centers;
  }, [nodes]);

  const minimapOutgoingConnections = useMemo(() => {
    const outgoing = new Map<string, Array<{ id: string; x: number; y: number }>>();

    edges.forEach((edge) => {
      const targetCenter = minimapNodeCenters.get(edge.target);
      if (!targetCenter) {
        return;
      }

      const existing = outgoing.get(edge.source) ?? [];
      existing.push({
        id: edge.id,
        x: targetCenter.x,
        y: targetCenter.y,
      });
      outgoing.set(edge.source, existing);
    });

    return outgoing;
  }, [edges, minimapNodeCenters]);

  const MiniMapNodeWithConnections = useCallback(
    (nodeProps: MiniMapNodeProps) => {
      const {
        id,
        x,
        y,
        width,
        height,
        borderRadius,
        className,
        color,
        shapeRendering,
        strokeColor,
        strokeWidth,
        style,
        onClick,
      } = nodeProps;
      const centerX = x + width / 2;
      const centerY = y + height / 2;
      const outgoing = minimapOutgoingConnections.get(id) ?? [];

      return (
        <g>
          {outgoing.map((connection) => (
            <line
              key={connection.id}
              x1={centerX}
              y1={centerY}
              x2={connection.x}
              y2={connection.y}
              stroke="#1F2937"
              strokeOpacity={0.65}
              strokeWidth={20}
              strokeLinecap="round"
              shapeRendering="geometricPrecision"
            />
          ))}
          <rect
            x={x}
            y={y}
            width={width}
            height={height}
            rx={borderRadius}
            ry={borderRadius}
            className={className}
            fill={color}
            stroke={strokeColor}
            strokeWidth={strokeWidth}
            shapeRendering={shapeRendering}
            style={style}
            onClick={(event) => onClick?.(event, id)}
          />
        </g>
      );
    },
    [minimapOutgoingConnections],
  );

  if (!isVisible) {
    return null;
  }

  return (
    <MiniMap
      position="bottom-left"
      pannable
      zoomable
      bgColor="#F8FAFC"
      maskColor="rgba(255, 255, 255, 0.26)"
      maskStrokeColor="rgba(100, 116, 139, 0.5)"
      maskStrokeWidth={1}
      offsetScale={0}
      nodeColor="#1F2937"
      nodeStrokeColor="transparent"
      nodeBorderRadius={8}
      nodeComponent={MiniMapNodeWithConnections}
      className="sp-canvas-minimap !bg-white !border-0 !outline !outline-1 !outline-slate-950/20 !rounded-md !overflow-hidden !shadow-none"
      style={{ marginBottom: 56, marginLeft: 16, width: MINIMAP_WIDTH, height: MINIMAP_HEIGHT }}
    />
  );
}
