type ScreenPoint = {
  x: number;
  y: number;
};

type DropdownPositionInput = {
  cursor: ScreenPoint;
  viewportWidth: number;
  viewportHeight: number;
  cursorLineHeight: number;
  dropdownWidth: number;
  dropdownHeight: number;
  valuePreviewWidth: number;
  showValuePreview: boolean;
  edgePadding?: number;
  gap?: number;
};

export function calculateDropdownPosition({
  cursor,
  viewportWidth,
  viewportHeight,
  cursorLineHeight,
  dropdownWidth,
  dropdownHeight,
  valuePreviewWidth,
  showValuePreview,
  edgePadding = 16,
  gap = 4,
}: DropdownPositionInput) {
  // Horizontal flip calculation
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

  // Vertical flip calculation
  let top = cursor.y + gap;
  const spaceBelow = viewportHeight - top - edgePadding;
  
  if (spaceBelow < dropdownHeight) {
    // If we don't have enough space below, check space above the cursor text
    const spaceAbove = cursor.y - cursorLineHeight - gap - edgePadding;
    
    // Flip above if there's more space above than below, or if space above can fully fit the dropdown
    if (spaceAbove > spaceBelow || spaceAbove >= dropdownHeight) {
      top = cursor.y - cursorLineHeight - gap - dropdownHeight;
    }
  }

  // Ensure top is never rendered off-screen (if viewport is too small)
  top = Math.max(edgePadding, Math.min(top, viewportHeight - dropdownHeight - edgePadding));

  return { top, left };
}
