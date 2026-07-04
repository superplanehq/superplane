import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";

export function InstallShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen flex flex-col bg-slate-100 dark:bg-slate-900">
      <header className="flex h-12 items-center border-b border-slate-950/15 bg-white px-4 dark:border-gray-800 dark:bg-gray-900">
        <OrganizationMenuButton />
      </header>
      <main className="flex w-full flex-grow-1 flex-col">
        <div className="mx-auto w-full max-w-[640px] flex-grow-1 p-8">{children}</div>
      </main>
    </div>
  );
}
