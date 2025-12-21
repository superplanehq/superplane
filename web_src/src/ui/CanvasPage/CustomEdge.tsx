import { CSSProperties, useCallback } from "react";
import { BaseEdge, EdgeLabelRenderer, EdgeProps, getBezierPath, useReactFlow } from "@xyflow/react";
import { CircleX } from "lucide-react";

export function CustomEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  style = {},
  selected,
  data,
}: EdgeProps) {
  const { setEdges } = useReactFlow();
  const isHovered = data?.isHovered || false;
  const onDeleteEdge = data?.onDelete as ((edgeId: string) => void) | undefined;

  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  const handleEdgeDelete = useCallback(() => {
    if (onDeleteEdge) {
      onDeleteEdge(id);
      return;
    }

    setEdges((edges) => edges.filter((edge) => edge.id !== id));
  }, [id, onDeleteEdge, setEdges]);

  // Update style based on selection and hover state
  const edgeStyle: CSSProperties = {
    ...style,
    stroke: selected || isHovered ? "#A1AEC0" : style.stroke || "#DEF3FE",
    strokeWidth: selected ? 3 : style.strokeWidth || 3,
    pointerEvents: "visibleStroke",
  };
  const shouldShowIcon = isHovered || selected;

  return (
    <>
      <BaseEdge path={edgePath} style={edgeStyle} interactionWidth={20} className={isHovered ? "hovered" : undefined} />
      <path
        d={edgePath}
        fill="none"
        stroke="transparent"
        strokeWidth={20}
        style={{ cursor: "pointer", pointerEvents: "stroke" }}
        onPointerDown={(event) => {
          if (event.button !== 0) return;
          event.stopPropagation();
          handleEdgeDelete();
        }}
      />
      <EdgeLabelRenderer>
        <div
          style={{
            position: "absolute",
            transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
            width: "32px",
            height: "32px",
            zIndex: 1001,
            pointerEvents: "none",
            opacity: shouldShowIcon ? 1 : 0,
            transition: "opacity 150ms ease",
          }}
          className="edge-label nodrag nopan flex items-center justify-center"
        >
          <div className="rounded-full bg-slate-100 p-1">
            <CircleX size={18} className="text-slate-500" />
          </div>
        </div>
      </EdgeLabelRenderer>
    </>
  );
}
