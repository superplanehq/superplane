import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { Link } from "@/components/Link/link";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import { ChevronLeft } from "lucide-react";
import type { ReactNode } from "react";
import { useParams } from "react-router-dom";

const pageShellClassName = cn("min-h-screen flex flex-col bg-slate-100", appDarkModeClasses.surface);

const pageHeaderClassName = cn(
  "flex h-10 items-center gap-2 border-b bg-white px-2 sm:px-3",
  appDarkModeClasses.sidebarEdge,
  appDarkModeClasses.surface,
);

const pageContentClassName = cn("w-full flex-grow-1 bg-slate-100", appDarkModeClasses.surface);

interface FactoryPageShellProps {
  children: ReactNode;
  backHref?: string;
  backLabel?: string;
}

export function FactoryPageShell({ children, backHref, backLabel }: FactoryPageShellProps) {
  const { organizationId } = useParams<{ organizationId: string }>();

  if (!organizationId) {
    return null;
  }

  return (
    <div className={pageShellClassName}>
      <header className={pageHeaderClassName}>
        <OrganizationMenuButton organizationId={organizationId} />
        {backHref ? (
          <Link
            href={backHref}
            className="inline-flex items-center gap-1 text-sm font-medium text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100"
          >
            <ChevronLeft className="h-4 w-4" aria-hidden />
            {backLabel ?? "Back"}
          </Link>
        ) : null}
      </header>
      <main className="w-full h-full flex flex-column flex-grow-1">
        <div className={pageContentClassName}>{children}</div>
      </main>
    </div>
  );
}
