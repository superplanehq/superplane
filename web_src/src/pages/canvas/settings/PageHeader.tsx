import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";

type PageHeaderProps = {
  organizationId: string;
  title: string;
};

export function PageHeader({ organizationId, title }: PageHeaderProps) {
  return (
    <header
      className={cn(
        "relative flex h-11 shrink-0 items-center border-b bg-white px-3 sm:px-4",
        appDarkModeClasses.surface,
        appDarkModeClasses.sidebarEdge,
      )}
    >
      <div className="relative z-10 flex min-w-0 shrink-0 items-center">
        <OrganizationMenuButton organizationId={organizationId} />
      </div>

      <div className="pointer-events-none absolute inset-x-0 flex justify-center px-24">
        <span className={cn("truncate text-center text-sm font-medium text-slate-900", appDarkModeClasses.textPrimary)}>
          {title}
        </span>
      </div>
      <div className="relative z-10 ml-auto w-9 shrink-0" aria-hidden />
    </header>
  );
}
