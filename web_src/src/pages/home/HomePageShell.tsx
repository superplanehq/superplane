import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import type { ReactNode } from "react";
import { useParams } from "react-router-dom";

const pageShellClassName = cn("min-h-screen flex flex-col bg-slate-100", appDarkModeClasses.surface);

const pageHeaderClassName = cn(
  "flex h-10 items-center border-b bg-white px-2 sm:px-3",
  appDarkModeClasses.sidebarEdge,
  appDarkModeClasses.surface,
);

const pageContentClassName = cn("w-full flex-grow-1 bg-slate-100", appDarkModeClasses.surface);

export function HomePageShell({ children }: { children: ReactNode }) {
  const { organizationId } = useParams<{ organizationId: string }>();

  if (!organizationId) {
    return null;
  }

  return (
    <div className={pageShellClassName}>
      <header className={pageHeaderClassName}>
        <OrganizationMenuButton organizationId={organizationId} />
      </header>
      <main className="w-full h-full flex flex-column flex-grow-1">
        <div className={pageContentClassName}>{children}</div>
      </main>
    </div>
  );
}
