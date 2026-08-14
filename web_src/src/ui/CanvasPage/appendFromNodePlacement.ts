import {
  FACTORY_NODE_CARD_HEIGHT,
  FACTORY_NODE_CARD_WIDTH,
  FACTORY_NODE_VERTICAL_GAP,
} from "@/lib/factoryCanvasChrome";

type FlowPoint = { x: number; y: number };
type Viewport = { x: number; y: number; zoom: number };

type AppendPlacementInput = {
  sourcePosition: FlowPoint;
  sourceWidth: number;
  sourceHeight: number;
  isVerticalFlow: boolean;
  viewport: Viewport;
  canvasWidth: number;
  canvasHeight: number;
};

type AppendPlacementResult = {
  placeholderPosition: FlowPoint;
  nextViewport: Viewport | null;
};

const APPEND_GAP_X = 300;
const APPEND_GAP_Y = 220;
const APPEND_ALIGNMENT_Y = 30;
const APPEND_ALIGNMENT_X = 0;
const RIGHT_SIDEBAR_SAFE_AREA = 560;
const BOTTOM_SAFE_AREA = 160;
const PLACEHOLDER_ESTIMATED_WIDTH = 420;
const PLACEHOLDER_ESTIMATED_HEIGHT = 220;
const VIEWPORT_BUFFER = 48;

export function computeAppendFromNodePlacement({
  sourcePosition,
  sourceWidth,
  sourceHeight,
  isVerticalFlow,
  viewport,
  canvasWidth,
  canvasHeight,
}: AppendPlacementInput): AppendPlacementResult {
  const gapY = isVerticalFlow ? FACTORY_NODE_VERTICAL_GAP : APPEND_GAP_Y;
  const estimatedWidth = isVerticalFlow ? FACTORY_NODE_CARD_WIDTH : PLACEHOLDER_ESTIMATED_WIDTH;
  const estimatedHeight = isVerticalFlow ? FACTORY_NODE_CARD_HEIGHT : PLACEHOLDER_ESTIMATED_HEIGHT;

  const placeholderPosition = isVerticalFlow
    ? {
        x: sourcePosition.x + APPEND_ALIGNMENT_X,
        y: sourcePosition.y + sourceHeight + gapY,
      }
    : {
        x: sourcePosition.x + sourceWidth + APPEND_GAP_X,
        y: sourcePosition.y + APPEND_ALIGNMENT_Y,
      };

  if (!isVerticalFlow) {
    const placeholderRightScreenX = (placeholderPosition.x + estimatedWidth) * viewport.zoom + viewport.x;
    const maxVisibleScreenX = canvasWidth - RIGHT_SIDEBAR_SAFE_AREA - VIEWPORT_BUFFER;
    if (canvasWidth > 0 && placeholderRightScreenX > maxVisibleScreenX) {
      const overflow = placeholderRightScreenX - maxVisibleScreenX;
      return {
        placeholderPosition,
        nextViewport: { ...viewport, x: viewport.x - overflow },
      };
    }
    return { placeholderPosition, nextViewport: null };
  }

  const placeholderBottomScreenY = (placeholderPosition.y + estimatedHeight) * viewport.zoom + viewport.y;
  const maxVisibleScreenY = canvasHeight - BOTTOM_SAFE_AREA - VIEWPORT_BUFFER;
  if (canvasHeight > 0 && placeholderBottomScreenY > maxVisibleScreenY) {
    const overflow = placeholderBottomScreenY - maxVisibleScreenY;
    return {
      placeholderPosition,
      nextViewport: { ...viewport, y: viewport.y - overflow },
    };
  }
  return { placeholderPosition, nextViewport: null };
}
