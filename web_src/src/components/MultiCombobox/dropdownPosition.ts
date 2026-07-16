export interface DropdownRect {
  top: number;
  bottom: number;
  left: number;
  width: number;
}

export interface DropdownPosition {
  /** Distance from the top of the viewport, set when the dropdown opens downward. */
  top?: number;
  /** Distance from the bottom of the viewport, set when the dropdown opens upward. */
  bottom?: number;
  left: number;
  width: number;
  /** Height available for the dropdown so it never overflows the viewport. */
  maxHeight: number;
}

interface ComputeOptions {
  /** Space between the trigger and the dropdown. */
  gap?: number;
  /** Ideal dropdown height when there is enough room. */
  preferredMaxHeight?: number;
}

/**
 * Decides whether the dropdown should open below or above the trigger so it stays
 * within the viewport. It opens upward only when there is not enough room below and
 * more room above, matching the behaviour requested for config forms where the
 * select is the last field.
 */
export function computeDropdownPosition(
  rect: DropdownRect,
  viewportHeight: number,
  { gap = 4, preferredMaxHeight = 240 }: ComputeOptions = {},
): DropdownPosition {
  const spaceBelow = viewportHeight - rect.bottom - gap;
  const spaceAbove = rect.top - gap;
  const openUpward = spaceBelow < preferredMaxHeight && spaceAbove > spaceBelow;

  if (openUpward) {
    return {
      bottom: viewportHeight - rect.top + gap,
      left: rect.left,
      width: rect.width,
      maxHeight: Math.max(0, Math.min(preferredMaxHeight, spaceAbove)),
    };
  }

  return {
    top: rect.bottom + gap,
    left: rect.left,
    width: rect.width,
    maxHeight: Math.max(0, Math.min(preferredMaxHeight, spaceBelow)),
  };
}
