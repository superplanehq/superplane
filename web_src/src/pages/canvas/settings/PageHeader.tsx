import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";

type PageHeaderProps = {
  organizationId: string;
  /** Centered title (e.g. "My canvas · Settings"). Omit for minimal chrome (loading/error). */
  centerTitle?: string;
};

export function PageHeader({ organizationId, centerTitle }: PageHeaderProps) {
  return (
    <header className="relative flex h-11 shrink-0 items-center border-b border-slate-950/15 bg-white px-3 sm:px-4">
      <div className="relative z-10 flex min-w-0 shrink-0 items-center">
        <OrganizationMenuButton organizationId={organizationId} />
      </div>
      {centerTitle ? (
        <>
          <div className="pointer-events-none absolute inset-x-0 flex justify-center px-24">
            <span className="truncate text-center text-sm font-medium text-slate-900">{centerTitle}</span>
          </div>
          <div className="relative z-10 ml-auto w-9 shrink-0" aria-hidden />
        </>
      ) : null}
    </header>
  );
}
