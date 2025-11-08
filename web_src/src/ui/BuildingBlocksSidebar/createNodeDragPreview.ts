import type { BuildingBlock } from "./index";

/**
 * Creates a custom drag preview element that visually matches a canvas node.
 * The preview is scaled according to the current canvas zoom level.
 */
export function createNodeDragPreview(
  e: React.DragEvent<HTMLDivElement>,
  block: BuildingBlock,
  colorClass: string,
  backgroundColorClass: string,
  canvasZoom: number,
) {
  e.dataTransfer.effectAllowed = "move";
  e.dataTransfer.setData("application/reactflow", JSON.stringify(block));

  // Create a custom drag preview that looks like a node
  // Nodes use w-[26rem] (416px) for composite or w-[23rem] (368px) for others
  // Using 26rem (416px) as the standard size
  // Scale dimensions directly to match the canvas zoom level
  // With box-sizing: border-box, width includes border
  const baseWidth = 420; // 26rem + slight adjustment to match actual render size
  const baseBorderWidth = 2;
  const scaledWidth = baseWidth * canvasZoom;
  const baseIconSize = 24; // 1.5rem
  const scaledIconSize = baseIconSize * canvasZoom;
  const baseFontSize = 18; // 1.125rem
  const scaledFontSize = baseFontSize * canvasZoom;
  const basePadding = 8; // 0.5rem
  const scaledPadding = basePadding * canvasZoom;
  const baseGap = 8; // 0.5rem
  const scaledGap = baseGap * canvasZoom;
  const baseBorderRadius = 6;
  const scaledBorderRadius = baseBorderRadius * canvasZoom;
  const scaledBorderWidth = Math.max(1, baseBorderWidth * canvasZoom);

  const dragPreview = document.createElement("div");
  dragPreview.style.cssText = `
    position: absolute;
    top: -1000px;
    left: -1000px;
    width: ${scaledWidth}px;
    background: white;
    border: ${scaledBorderWidth}px solid #e5e7eb;
    border-radius: ${scaledBorderRadius}px;
    box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
    pointer-events: none;
    z-index: 9999;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    overflow: hidden;
    box-sizing: border-box;
  `;

  // Create the icon container with background color (matches ComponentHeader w-6 h-6)
  const iconContainer = document.createElement("div");
  iconContainer.style.cssText = `
    width: ${scaledIconSize}px;
    height: ${scaledIconSize}px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 9999px;
    flex-shrink: 0;
    overflow: hidden;
  `;
  iconContainer.className = backgroundColorClass;

  // Create the icon element (matches ComponentHeader size={20})
  const iconSvgSize = 20 * canvasZoom;
  const iconWrapper = document.createElement("div");
  iconWrapper.className = colorClass;
  iconWrapper.innerHTML = `
    <svg width="${iconSvgSize}" height="${iconSvgSize}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <circle cx="12" cy="12" r="10"></circle>
    </svg>
  `;
  iconContainer.appendChild(iconWrapper);

  // Create the content (matches ComponentHeader text-lg font-semibold)
  const content = document.createElement("div");
  content.style.cssText = `flex: 1; min-width: 0;`;
  content.innerHTML = `
    <div style="font-weight: 600; font-size: ${scaledFontSize}px; line-height: ${scaledFontSize * 1.5}px; color: #18181b; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">${block.label || block.name}</div>
  `;

  // Assemble the preview (matches ComponentHeader structure with header styling)
  const header = document.createElement("div");
  header.style.cssText = `
    display: flex;
    align-items: center;
    gap: ${scaledGap}px;
    padding: ${scaledPadding}px;
    background: #f9fafb;
    border-bottom: ${scaledBorderWidth}px solid #e5e7eb;
  `;
  header.appendChild(iconContainer);
  header.appendChild(content);

  // Add a body section to make it look more like a complete node
  const bodyPadding = 12 * canvasZoom; // 0.75rem
  const bodyMinHeight = 80 * canvasZoom; // 5rem
  const body = document.createElement("div");
  body.style.cssText = `
    padding: ${bodyPadding}px ${scaledPadding}px;
    min-height: ${bodyMinHeight}px;
    background: white;
  `;

  dragPreview.appendChild(header);
  dragPreview.appendChild(body);

  document.body.appendChild(dragPreview);
  // Center the drag preview on cursor
  // scaledWidth already includes border due to border-box
  const offsetX = scaledWidth / 2;
  const offsetY = 30 * canvasZoom;
  e.dataTransfer.setDragImage(dragPreview, offsetX, offsetY);

  // Clean up the preview element after drag starts
  setTimeout(() => {
    document.body.removeChild(dragPreview);
  }, 0);
}
