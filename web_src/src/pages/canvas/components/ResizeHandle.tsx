interface ResizeHandleProps {
  onMouseDown: () => void;
  onMouseEnter: () => void;
  onMouseLeave: () => void;
}

export const ResizeHandle = ({
  onMouseDown,
  onMouseEnter,
  onMouseLeave
}: ResizeHandleProps) => {
  return (
    <div
      className={`absolute left-0 top-0 bottom-0 w-2 cursor-ew-resize rounded transition-colors`}
      style={{ zIndex: 100 }}
      onMouseDown={onMouseDown}
      onMouseEnter={onMouseEnter}
      onMouseLeave={onMouseLeave}
    />
  );
};