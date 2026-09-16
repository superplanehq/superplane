type ScreenPoint = {
  x: number;
  y: number;
};

type DropdownPositionInput = {
  cursor: ScreenPoint;
  inputTop?: number;
  inputBottom?: number;
  viewportWidth: number;
  viewportHeight?: number;
  dropdownWidth: number;
  valuePreviewWidth: number;
  showValuePreview: boolean;
  dropdownHeight?: number;
  edgePadding?: number;
  gap?: number;
};

export function calculateDropdownPosition({
  cursor,
  viewportWidth,
  viewportHeight = typeof window !== "undefined" ? window.innerHeight : 800,
  dropdownWidth,
  valuePreviewWidth,
  showValuePreview,
  dropdownHeight = 244,
  edgePadding = 16,
  gap = 4,
}: DropdownPositionInput) {
  const totalWidth = showValuePreview ? dropdownWidth + valuePreviewWidth : dropdownWidth;

  // Horizontal positioning (flip left if not enough space on right)
  const spaceOnRight = viewportWidth - cursor.x - edgePadding;
  const spaceOnLeft = cursor.x - edgePadding;
  const shouldFlipLeft = spaceOnRight < dropdownWidth && spaceOnLeft >= dropdownWidth;

  let left: number;
  if (shouldFlipLeft) {
    left = showValuePreview ? cursor.x - dropdownWidth - valuePreviewWidth : cursor.x - dropdownWidth;
  } else {
    left = showValuePreview ? cursor.x - valuePreviewWidth : cursor.x;
  }

  // Vertical positioning anchored to cursor y coordinate:
  // Position below cursor by default, or flip above cursor if space below is insufficient.
  const cursorLineHeight = 24;
  const spaceBelow = viewportHeight - cursor.y - gap - edgePadding;
  const spaceAbove = cursor.y - cursorLineHeight - gap - edgePadding;

  let top: number;
  if (spaceBelow < dropdownHeight && spaceAbove >= dropdownHeight) {
    // Flip above cursor line
    top = cursor.y - cursorLineHeight - dropdownHeight - gap;
  } else {
    // Position below cursor line
    top = cursor.y + gap;
  }

  // Ensure top stays within viewport edge boundaries
  const maxTop = Math.max(edgePadding, viewportHeight - dropdownHeight - edgePadding);
  top = Math.max(edgePadding, Math.min(top, maxTop));

  return {
    top,
    left: Math.max(edgePadding, Math.min(left, viewportWidth - totalWidth - edgePadding)),
  };
}
