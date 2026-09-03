import type { ReactNode } from "react";

export function WelcomeSurveyLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4 py-10">
      <div className="w-full max-w-xl rounded-lg border border-border bg-card px-8 py-9 text-foreground shadow-sm sm:px-10">
        {children}
      </div>
    </div>
  );
}
