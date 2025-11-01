import { useCallback } from 'react';
import {
  BaseEdge,
  EdgeLabelRenderer,
  EdgeProps,
  getBezierPath,
  useReactFlow,
} from '@xyflow/react';
import { CircleX } from 'lucide-react';

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
}: EdgeProps) {
  const { setEdges } = useReactFlow();

  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  const onEdgeClick = useCallback(() => {
    setEdges((edges) =>
      edges.map((edge) => ({
        ...edge,
        selected: edge.id === id,
      }))
    );
  }, [id, setEdges]);

  const onDeleteClick = useCallback(
    (event: React.MouseEvent) => {
      event.stopPropagation();
      setEdges((edges) => edges.filter((edge) => edge.id !== id));
    },
    [id, setEdges]
  );

  // Update style based on selection state
  const edgeStyle = {
    ...style,
    stroke: selected ? '#3B82F6' : style.stroke || '#C9D5E1',
    strokeWidth: selected ? 3 : style.strokeWidth || 3,
  };

  return (
    <>
      <BaseEdge
        path={edgePath}
        style={edgeStyle}
        onClick={onEdgeClick}
      />
      <EdgeLabelRenderer>
        {selected && (
          <div
            style={{
              position: 'absolute',
              transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
              pointerEvents: 'all',
            }}
            className="nodrag nopan"
          >
            <button
              className="flex items-center justify-center bg-white rounded-full shadow-lg hover:bg-red-50 transition-colors cursor-pointer"
              onClick={onDeleteClick}
              aria-label="Delete edge"
            >
              <CircleX size={20} className="text-red-500" />
            </button>
          </div>
        )}
      </EdgeLabelRenderer>
    </>
  );
}
