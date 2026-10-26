import { useCallback, useState } from "react";

/** A yes/no setup choice that survives reloads and integration redirects. */
function usePersistedOnboardingChoice(key: string | null, fallback: boolean) {
  const [value, setValue] = useState(() => {
    try {
      const stored = key ? sessionStorage.getItem(key) : null;
      return stored === null ? fallback : stored === "true";
    } catch {
      return fallback;
    }
  });
  const select = useCallback(
    (next: boolean) => {
      try {
        if (key) sessionStorage.setItem(key, String(next));
      } catch {
        // Blocked or full storage must not prevent an in-memory selection.
      }
      setValue(next);
    },
    [key],
  );
  return [value, select] as const;
}

/** Keep the import choice through reloads and integration redirects. */
export function useOnboardingInitialImportChoice(options?: { persistInitialImportKey?: string }) {
  const factoryId = options?.persistInitialImportKey;
  return usePersistedOnboardingChoice(
    factoryId ? `superplane:onboarding-skip-initial-import:${factoryId}` : null,
    false,
  );
}

/** Keep the automatic intake choice through reloads and integration redirects. Defaults to on. */
export function useOnboardingAutomaticIntakeChoice(options?: { persistInitialImportKey?: string }) {
  const factoryId = options?.persistInitialImportKey;
  return usePersistedOnboardingChoice(factoryId ? `superplane:onboarding-automatic-intake:${factoryId}` : null, true);
}
