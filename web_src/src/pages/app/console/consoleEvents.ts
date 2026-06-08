/**
 * Window-level CustomEvent names used as a fallback when no direct callback
 * has been wired into the {@link ConsoleContextProvider}. Consumers (e.g.
 * the canvas page host) can listen for these and route them into the same
 * flows that drive native trigger / approve / cancel actions.
 */
export const CONSOLE_TRIGGER_NODE_EVENT = "console:trigger-node";
export const CONSOLE_EXECUTION_ACTION_EVENT = "console:execution-action";
