import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";

export function InstallShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col bg-surface-canvas">
      <header className="flex h-12 items-center border-b border-edge-default bg-surface-default px-4">
        <OrganizationMenuButton />
      </header>
      <main className="flex w-full flex-grow-1 flex-col">
        <div className="mx-auto w-full max-w-[640px] flex-grow-1 p-8">{children}</div>
      </main>
    </div>
  );
}
