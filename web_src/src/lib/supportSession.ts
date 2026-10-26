export const SUPPORT_SESSION_READ_ONLY_MESSAGE = "This is a read-only support session. You cannot make changes.";
export const SUPPORT_SESSION_START_PATH = "/admin/api/support-session/start";
export const SUPPORT_SESSION_END_PATH = "/admin/api/support-session/end";

const READ_METHODS = new Set(["GET", "HEAD", "OPTIONS"]);

// Keep in sync with supportSessionAllowedWrites in pkg/public/middleware/support_session.go.
const ALLOWED_WRITE_PATHS = [
  /^\/admin\/api\/support-session\/end$/,
  /^\/api\/v1\/factories\/[^/]+\/scheduled-intake\/preview$/,
  /^\/admin\/api\/installation\/llm-providers\/[^/]+\/models$/,
];

let supportSessionActive = false;

export function setSupportSessionActive(active: boolean): void {
  supportSessionActive = active;
}

export function isSupportSessionActive(): boolean {
  return supportSessionActive;
}

export function isSupportSessionAllowedRequest(method: string, path: string): boolean {
  if (READ_METHODS.has(method.toUpperCase())) {
    return true;
  }
  return ALLOWED_WRITE_PATHS.some((pattern) => pattern.test(path));
}
