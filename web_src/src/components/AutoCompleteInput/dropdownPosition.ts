type ScreenPoint = {
  x: number;
  y: number;
};

type DropdownPositionInput = {
  cursor: ScreenPoint;
  /** Top edge of the cursor character in viewport coordinates (for flip-above). */
  cursorTop: number;
  viewportWidth: number;
  viewportHeight: number;
  dropdownWidth: number;
  dropdownHeight: number;
  valuePreviewWidth: number;
  showValuePreview: boolean;
  edgePadding?: number;
  gap?: number;
};

export function calculateDropdownPosition({
  cursor,
  cursorTop,
  viewportWidth,
  viewportHeight,
  dropdownWidth,
  dropdownHeight,
  valuePreviewWidth,
  showValuePreview,
  edgePadding = 16,
  gap = 4,
}: DropdownPositionInput) {
  // Horizontal placement
  const spaceOnRight = viewportWidth - cursor.x - edgePadding;
  const spaceOnLeft = cursor.x - edgePadding;
  const shouldFlipLeft = spaceOnRight < dropdownWidth && spaceOnLeft >= dropdownWidth;

  let left: number;
  if (shouldFlipLeft) {
    left = showValuePreview ? cursor.x - dropdownWidth - valuePreviewWidth : cursor.x - dropdownWidth;
  } else {
    left = showValuePreview ? cursor.x - valuePreviewWidth : cursor.x;
  }

  const totalWidth = showValuePreview ? dropdownWidth + valuePreviewWidth : dropdownWidth;
  left = Math.max(edgePadding, Math.min(left, viewportWidth - totalWidth - edgePadding));

  // Vertical placement — flip above the cursor when there isn't enough room below
  const spaceBelow = viewportHeight - cursor.y - edgePadding;
  const spaceAbove = cursorTop - edgePadding;
  const shouldFlipAbove = spaceBelow < dropdownHeight && spaceAbove >= dropdownHeight;

  const top = shouldFlipAbove
    ? Math.max(edgePadding, cursorTop - gap - dropdownHeight)
    : Math.min(cursor.y + gap, viewportHeight - dropdownHeight - edgePadding);

  return { top, left };
}
