import { useCallback, useState } from "react";

/** Keep the import choice through reloads and integration redirects. */
export function useOnboardingInitialImportChoice(options?: { persistInitialImportKey?: string }) {
  const factoryId = options?.persistInitialImportKey;
  const key = factoryId ? `superplane:onboarding-skip-initial-import:${factoryId}` : null;
  const [skipInitialImport, setSkipInitialImport] = useState(() => {
    try {
      return key ? sessionStorage.getItem(key) === "true" : false;
    } catch {
      return false;
    }
  });
  const selectInitialImport = useCallback(
    (skip: boolean) => {
      try {
        if (key) sessionStorage.setItem(key, String(skip));
      } catch {
        // Blocked or full storage must not prevent an in-memory selection.
      }
      setSkipInitialImport(skip);
    },
    [key],
  );
  return [skipInitialImport, selectInitialImport] as const;
}
