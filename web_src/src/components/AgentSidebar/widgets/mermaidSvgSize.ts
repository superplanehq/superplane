export function normalizeSvgForViewport(svgEl: SVGSVGElement) {
  svgEl.style.maxWidth = "none";
  svgEl.style.height = "auto";

  const bounds = readSvgSize(svgEl);
  if (bounds.width > 0) {
    svgEl.setAttribute("width", String(bounds.width));
  }
  if (bounds.height > 0) {
    svgEl.setAttribute("height", String(bounds.height));
  }
}

function parseSvgLengthAttr(value: string | null): number {
  if (!value || value.endsWith("%")) {
    return NaN;
  }
  return Number.parseFloat(value);
}

export function readSvgSize(svgEl: SVGSVGElement): { width: number; height: number } {
  const viewBox = svgEl.viewBox?.baseVal;
  if (viewBox && viewBox.width > 0 && viewBox.height > 0) {
    return { width: viewBox.width, height: viewBox.height };
  }

  try {
    const bbox = svgEl.getBBox();
    if (bbox.width > 0 && bbox.height > 0) {
      return { width: bbox.width, height: bbox.height };
    }
  } catch {
    // getBBox can throw if the SVG is not rendered yet.
  }

  const width = parseSvgLengthAttr(svgEl.getAttribute("width"));
  const height = parseSvgLengthAttr(svgEl.getAttribute("height"));
  return {
    width: Number.isFinite(width) && width > 0 ? width : svgEl.clientWidth,
    height: Number.isFinite(height) && height > 0 ? height : svgEl.clientHeight,
  };
}

export function computeFitScale(
  viewportWidth: number,
  viewportHeight: number,
  contentWidth: number,
  contentHeight: number,
  maxScale = 5,
): number {
  if (contentWidth <= 0 || contentHeight <= 0) {
    return 1;
  }
  return Math.min(viewportWidth / contentWidth, viewportHeight / contentHeight, maxScale);
}
