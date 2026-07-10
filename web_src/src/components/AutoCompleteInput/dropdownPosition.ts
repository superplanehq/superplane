type ScreenPoint = {
  x: number;
  y: number;
};

type DropdownPositionInput = {
  cursor: ScreenPoint;
  viewportWidth: number;
  dropdownWidth: number;
  valuePreviewWidth: number;
  showValuePreview: boolean;
  edgePadding?: number;
  gap?: number;
};

export function calculateDropdownPosition({
  cursor,
  viewportWidth,
  dropdownWidth,
  valuePreviewWidth,
  showValuePreview,
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

  const totalWidth = showValuePreview ? dropdownWidth + valuePreviewWidth : dropdownWidth;
  return {
    top: cursor.y + gap,
    left: Math.max(edgePadding, Math.min(left, viewportWidth - totalWidth - edgePadding)),
  };
}
