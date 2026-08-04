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
  edgePadding = 16,
  gap = 4,
}: DropdownPositionInput) {
  const spaceOnRight = viewportWidth - cursor.x - edgePadding;
  const spaceOnLeft = cursor.x - edgePadding;

  const shouldFlipLeft =
    spaceOnRight < dropdownWidth && spaceOnLeft >= dropdownWidth;

  let left: number;

  if (shouldFlipLeft) {
    left = showValuePreview
      ? cursor.x - dropdownWidth - valuePreviewWidth
      : cursor.x - dropdownWidth;
  } else {
    left = showValuePreview ? cursor.x - valuePreviewWidth : cursor.x;
  }

  const totalWidth = showValuePreview
    ? dropdownWidth + valuePreviewWidth
    : dropdownWidth;

  const spaceBelow = viewportHeight - cursor.y - edgePadding;
  const spaceAbove = cursor.y - edgePadding;

  const shouldFlipUp =
    spaceBelow < dropdownHeight && spaceAbove >= dropdownHeight;

  const top = shouldFlipUp
    ? cursor.y - dropdownHeight - gap
    : cursor.y + gap;

  return {
    top,
    left: Math.max(
      edgePadding,
      Math.min(left, viewportWidth - totalWidth - edgePadding),
    ),
  };
}