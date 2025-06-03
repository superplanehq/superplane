import Elk from "elkjs";

export const elk = new Elk({
  defaultLayoutOptions: {
    "elk.algorithm": "layered",
    "elk.direction": "RIGHT",
    "elk.layered.spacing": "80",
    "elk.layered.mergeEdges": "true",
    "elk.spacing": "80",
    "elk.spacing.individual": "80",
    "elk.edgeRouting": "SPLINES",
    "elk.layered.spacing.nodeNodeBetweenLayers": "100",
    "elk.spacing.nodeNode": "100",
  },
});