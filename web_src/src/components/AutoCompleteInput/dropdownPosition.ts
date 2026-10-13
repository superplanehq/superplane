type ScreenPoint = {
  x: number;
  y: number;
};

type DropdownPositionInput = {
  cursor: ScreenPoint;
  viewportWidth: number;
  viewportHeight: number;
  dropdownWidth: number;
  dropdownHeight: number;
  valuePreviewWidth: number;
  showValuePreview: boolean;
  cursorHeight?: number;
  edgePadding?: number;
  gap?: number;
};

export function calculateDropdownPosition({
  cursor,
  viewportWidth,
  viewportHeight,
  dropdownWidth,
  dropdownHeight,
  valuePreviewWidth,
  showValuePreview,
  cursorHeight = 0,
  edgePadding = 16,
  gap = 4,
}: DropdownPositionInput) {
  const spaceOnRight = viewportWidth - cursor.x - edgePadding;
  const spaceOnLeft = cursor.x - edgePadding;
  const shouldFlipLeft = spaceOnRight < dropdownWidth && spaceOnLeft >= dropdownWidth;

  let left: number;
  if (shouldFlipLeft) {
    left = showValuePreview ? cursor.x - dropdownWidth - valuePreviewWidth : cursor.x - dropdownWidth;
  } else {
    left = showValuePreview ? cursor.x - valuePreviewWidth : cursor.x;
  }

  // Flip the dropdown above the caret when there isn't enough room below it and
  // the space above is larger. `cursor.y` is the bottom of the caret, so we
  // clear the caret's own line (`cursorHeight`) before stacking upwards.
  const spaceBelow = viewportHeight - cursor.y - edgePadding;
  const spaceAbove = cursor.y - cursorHeight - edgePadding;
  const shouldFlipUp = spaceBelow < dropdownHeight && spaceAbove > spaceBelow;

  const top = shouldFlipUp ? cursor.y - cursorHeight - gap - dropdownHeight : cursor.y + gap;

  const totalWidth = showValuePreview ? dropdownWidth + valuePreviewWidth : dropdownWidth;
  return {
    top: Math.max(edgePadding, Math.min(top, viewportHeight - dropdownHeight - edgePadding)),
    left: Math.max(edgePadding, Math.min(left, viewportWidth - totalWidth - edgePadding)),
  };
}
