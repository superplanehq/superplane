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
    "elk.layered.spacing.nodeNodeBetweenLayers": "250",
    "elk.spacing.nodeNode": "100",
    "elk.separateConnectedComponents": "true",
    "elk.spacing.componentComponent": "50",
    "elk.componentArrangement": "TOPDOWN",
  },
});