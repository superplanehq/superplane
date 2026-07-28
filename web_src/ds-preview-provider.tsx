// Preview provider for design-sync (cfg.provider).
//
// Distilled from .storybook/preview.tsx. The converter normally bundles those
// decorators automatically, but this repo's preview.tsx calls MSW's
// initialize(), and the converter's inert msw stub makes worker.start() return
// a non-thenable — which crashed every preview with
// "worker.start(...).then is not a function". Setting cfg.provider skips
// decorator bundling entirely, so this file must carry the real context.
//
// Kept in sync by hand with .storybook/preview.tsx — see .design-sync/NOTES.md
// ("Re-sync risks"). MSW is deliberately NOT reproduced: previews render static
// stories, and any story needing mocked network is skipped via cfg.overrides.
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useLayoutEffect, useMemo, type ReactNode } from "react";

import { ThemeContext } from "./src/contexts/themeContextState";
import { applyResolvedThemeToDocument } from "./src/lib/themePreference";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      staleTime: Infinity,
    },
  },
});

/** Material Symbols, injected the same way .storybook/preview.tsx does it. */
function useMaterialSymbols() {
  useLayoutEffect(() => {
    const href = "https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined";
    if (document.querySelector(`link[href="${href}"]`)) return;
    const link = document.createElement("link");
    link.href = href;
    link.rel = "stylesheet";
    document.head.appendChild(link);
  }, []);
}

export function DsProvider({ children }: { children: ReactNode }) {
  useMaterialSymbols();
  useLayoutEffect(() => {
    applyResolvedThemeToDocument("light");
  }, []);

  const value = useMemo(
    () => ({
      preference: "light" as const,
      resolvedTheme: "light" as const,
      setPreference: () => {
        // Theme is fixed for previews, as it is in the Storybook shell.
      },
    }),
    [],
  );

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
    </QueryClientProvider>
  );
}
