import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import type { ReactNode } from "react";
import { useParams } from "react-router-dom";

export function HomePageShell({ children }: { children: ReactNode }) {
  const { organizationId } = useParams<{ organizationId: string }>();

  if (!organizationId) {
    return null;
  }

  return (
    <div className="min-h-screen flex flex-col bg-slate-100 dark:bg-slate-900">
      <header className="flex h-10 items-center border-b border-slate-950/15 bg-white px-2 sm:px-3">
        <OrganizationMenuButton organizationId={organizationId} />
      </header>
      <main className="w-full h-full flex flex-column flex-grow-1">
        <div className="bg-slate-100 w-full flex-grow-1">{children}</div>
      </main>
    </div>
  );
}
