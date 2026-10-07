import {
  FACTORY_NODE_CARD_HEIGHT,
  FACTORY_NODE_EDIT_VERTICAL_GAP,
  FACTORY_NODE_VERTICAL_GAP,
} from "./factoryCanvasChrome";

/** Same-layer nodes can differ by a few pixels after packing. */
const LAYER_Y_TOLERANCE_PX = 8;

export const FACTORY_EDIT_VERTICAL_EXTRA_PER_LAYER = FACTORY_NODE_EDIT_VERTICAL_GAP - FACTORY_NODE_VERTICAL_GAP;

type PositionedNode = {
  position: { x: number; y: number };
};

function clusteredLayerYs(ys: number[]): number[] {
  const sorted = [...ys].sort((left, right) => left - right);
  const layers: number[] = [];
  for (const y of sorted) {
    const last = layers[layers.length - 1];
    if (last === undefined || y - last > LAYER_Y_TOLERANCE_PX) {
      layers.push(y);
    }
  }
  return layers;
}

function nearestLayerIndex(y: number, layers: number[]): number {
  let bestIndex = 0;
  let bestDistance = Number.POSITIVE_INFINITY;
  for (let index = 0; index < layers.length; index += 1) {
    const distance = Math.abs(y - layers[index]);
    if (distance < bestDistance) {
      bestDistance = distance;
      bestIndex = index;
    }
  }
  return bestIndex;
}

/**
 * Stretch stacked factory ranks in edit/configure only.
 * Spec / live / run keep compact ELK positions (FACTORY_NODE_VERTICAL_GAP).
 */
export function expandFactoryEditVerticalPositions<T extends PositionedNode>(nodes: T[]): T[] {
  if (nodes.length <= 1 || FACTORY_EDIT_VERTICAL_EXTRA_PER_LAYER === 0) {
    return nodes;
  }

  const layers = clusteredLayerYs(nodes.map((node) => node.position.y));
  if (layers.length <= 1) {
    return nodes;
  }

  return nodes.map((node) => {
    const rank = nearestLayerIndex(node.position.y, layers);
    if (rank === 0) {
      return node;
    }
    return {
      ...node,
      position: {
        x: node.position.x,
        y: node.position.y + rank * FACTORY_EDIT_VERTICAL_EXTRA_PER_LAYER,
      },
    };
  });
}

/** Compact layer stride used by factory ELK (card + view gap). */
export const FACTORY_COMPACT_LAYER_STRIDE = FACTORY_NODE_CARD_HEIGHT + FACTORY_NODE_VERTICAL_GAP;
