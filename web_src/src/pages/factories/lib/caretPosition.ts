export interface CaretCoordinates {
  top: number;
  left: number;
  height: number;
}

/** Textarea CSS properties that affect text layout/measurement and must be
 * copied onto the mirror so the marker lands where the real caret would. */
const MIRRORED_STYLE_PROPS = [
  "boxSizing",
  "fontFamily",
  "fontSize",
  "fontWeight",
  "fontStyle",
  "letterSpacing",
  "textTransform",
  "wordSpacing",
  "lineHeight",
  "paddingTop",
  "paddingRight",
  "paddingBottom",
  "paddingLeft",
  "borderTopWidth",
  "borderRightWidth",
  "borderBottomWidth",
  "borderLeftWidth",
  "tabSize",
  "wordBreak",
] as const;

/**
 * Compute the caret's viewport pixel position inside a `<textarea>` via the
 * standard "mirror div" technique: an offscreen div is given the same
 * box/font styles and fixed screen position as the textarea, filled with the
 * text up to `index`, and a zero-width marker span appended — its measured
 * position is where the real caret sits.
 */
export function getCaretCoordinates(textarea: HTMLTextAreaElement, index: number): CaretCoordinates {
  const rect = textarea.getBoundingClientRect();
  const computed = window.getComputedStyle(textarea);

  const mirror = document.createElement("div");
  mirror.style.position = "fixed";
  mirror.style.top = `${rect.top}px`;
  mirror.style.left = `${rect.left}px`;
  mirror.style.width = `${rect.width}px`;
  mirror.style.visibility = "hidden";
  mirror.style.pointerEvents = "none";
  mirror.style.whiteSpace = "pre-wrap";
  mirror.style.overflowWrap = "break-word";

  for (const prop of MIRRORED_STYLE_PROPS) {
    mirror.style[prop] = computed[prop];
  }

  mirror.textContent = textarea.value.slice(0, Math.max(0, index));

  const marker = document.createElement("span");
  marker.textContent = "​";
  mirror.appendChild(marker);

  document.body.appendChild(mirror);
  const markerRect = marker.getBoundingClientRect();
  document.body.removeChild(mirror);

  const lineHeight = parseFloat(computed.lineHeight);
  return {
    top: markerRect.top - textarea.scrollTop,
    left: markerRect.left - textarea.scrollLeft,
    height: markerRect.height || (Number.isFinite(lineHeight) ? lineHeight : 16),
  };
}
