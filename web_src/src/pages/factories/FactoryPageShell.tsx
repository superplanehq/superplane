import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { Link } from "@/components/Link/link";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import { ChevronLeft } from "lucide-react";
import type { ReactNode } from "react";
import { useParams } from "react-router-dom";
import { factoryPageBackgroundClassName } from "./factoryPageStyles";

const pageHeaderClassName = cn(
  "flex h-10 w-full shrink-0 items-center gap-2 border-b border-gray-200/80 bg-gray-50 px-2 sm:px-3 dark:border-gray-700/70",
  appDarkModeClasses.surface,
);

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
    <div className={cn("flex min-h-screen w-full min-w-0 flex-col", factoryPageBackgroundClassName)}>
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
      <main className="flex w-full min-w-0 flex-1 flex-col">
        <div className={cn(factoryPageBackgroundClassName, "w-full min-w-0 flex-1")}>{children}</div>
      </main>
    </div>
  );
}
