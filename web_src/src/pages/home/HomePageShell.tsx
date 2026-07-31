import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import type { ReactNode } from "react";
import { useParams } from "react-router-dom";

const pageShellClassName = "flex min-h-screen flex-col bg-surface-canvas";
const pageHeaderClassName = "flex h-10 items-center border-b border-edge-default bg-surface-default px-2 sm:px-3";
const pageContentClassName = "w-full flex-grow-1 bg-surface-canvas";

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
