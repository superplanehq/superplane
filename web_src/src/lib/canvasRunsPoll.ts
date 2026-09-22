export const CANVAS_RUNS_POLL_MS = 60_000;

export function canvasRunsPollInterval(websocketConnected: boolean): number | false {
  return websocketConnected ? false : CANVAS_RUNS_POLL_MS;
}
