/**
 * Window-level CustomEvent names used as a fallback when no direct callback
 * has been wired into the {@link DashboardContextProvider}. Consumers (e.g.
 * the canvas page host) can listen for these and route them into the same
 * flows that drive native trigger / approve / cancel actions.
 */
export const DASHBOARD_TRIGGER_NODE_EVENT = "dashboard:trigger-node";
export const DASHBOARD_EXECUTION_ACTION_EVENT = "dashboard:execution-action";
