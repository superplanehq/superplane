interface ResizeHandleProps {
  isDragging: boolean;
  onMouseDown: () => void;
  onMouseEnter: () => void;
  onMouseLeave: () => void;
}

export const ResizeHandle = ({ 
  isDragging, 
  onMouseDown, 
  onMouseEnter, 
  onMouseLeave 
}: ResizeHandleProps) => {
  return (
    <div
      className={`absolute left-0 top-0 bottom-0 w-2 cursor-ew-resize rounded transition-colors ${
        isDragging ? 'bg-gray-300' : 'bg-gray-200 hover:bg-gray-300'
      }`}
      style={{ zIndex: 100 }}
      onMouseDown={onMouseDown}
      onMouseEnter={onMouseEnter}
      onMouseLeave={onMouseLeave}
    />
  );
};