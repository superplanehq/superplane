import { cn } from "@/lib/utils";
import { TooltipProvider } from "@/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState, type ReactNode } from "react";

import { useFactoriesThemeClass } from "../../lib/useFactoriesThemeClass";

export function Shell({ children, className }: { children: ReactNode; className?: string }) {
  useFactoriesThemeClass();
  const [queryClient] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));

  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider delayDuration={150}>
        <div className={cn("w-full bg-background text-foreground", className)} data-testid="onboarding-shell">
          {children}
        </div>
      </TooltipProvider>
    </QueryClientProvider>
  );
}
