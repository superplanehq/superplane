export const LAST_USED_LOGIN_METHOD_STORAGE_KEY = "superplane:last-used-login-method";

export type LastUsedLoginMethod = "password" | "github" | "google";

const KNOWN_METHODS: LastUsedLoginMethod[] = ["password", "github", "google"];

function isKnownMethod(value: string | null): value is LastUsedLoginMethod {
  return value !== null && (KNOWN_METHODS as string[]).includes(value);
}

export function readLastUsedLoginMethod(): LastUsedLoginMethod | null {
  if (typeof window === "undefined") {
    return null;
  }

  try {
    const stored = window.localStorage.getItem(LAST_USED_LOGIN_METHOD_STORAGE_KEY);
    return isKnownMethod(stored) ? stored : null;
  } catch {
    return null;
  }
}

export function recordLastUsedLoginMethod(method: LastUsedLoginMethod): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(LAST_USED_LOGIN_METHOD_STORAGE_KEY, method);
  } catch {
    // Last-used hint persistence is optional.
  }
}
