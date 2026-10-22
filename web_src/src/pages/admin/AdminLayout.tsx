import SuperplaneLogo from "@/assets/superplane.svg";
import { Sidebar, SidebarBody, SidebarSection } from "@/components/Sidebar/sidebar";
import { Text } from "@/components/Text/text";
import { useAccount } from "@/contexts/useAccount";
import { useBillingProductEnabled } from "@/hooks/useBillingProductEnabled";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import { ArrowLeft, BookOpen, Building, KeyRound, Network, Shield, Terminal, Users, Webhook } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Link, Navigate, NavLink, Outlet } from "react-router";

type AdminPageLink = {
  label: string;
  to: string;
  icon: LucideIcon;
  end?: boolean;
};

const adminPageLinks: AdminPageLink[] = [
  { label: "Organizations", to: "/admin", icon: Building, end: true },
  { label: "Accounts", to: "/admin/accounts", icon: Users },
  { label: "Settings", to: "/admin/settings", icon: Network },
  { label: "Encryption", to: "/admin/encryption", icon: KeyRound },
  { label: "License", to: "/admin/license", icon: KeyRound },
  { label: "Billing plans", to: "/admin/billing-plans", icon: BookOpen },
  { label: "Features", to: "/admin/features", icon: BookOpen },
  { label: "Price Books", to: "/admin/price-books", icon: BookOpen },
  { label: "Fleets", to: "/admin/runner-tasks", icon: Terminal },
  { label: "Fleet setup", to: "/admin/fleet-manager-setup", icon: Terminal },
  { label: "Polar Webhooks", to: "/admin/polar-webhooks", icon: Webhook },
  { label: "Webhooks", to: "/admin/webhooks", icon: Webhook },
];

function adminSidebarNavLinkClass(active: boolean) {
  return cn(
    "group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium transition-colors",
    active
      ? "bg-sky-100 text-gray-800 dark:bg-gray-800 dark:text-gray-100"
      : "text-gray-500 hover:bg-sky-100 hover:text-gray-800 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100",
  );
}

function adminSidebarNavIconClass(active: boolean) {
  return cn(
    "text-gray-500 transition-colors group-hover:text-gray-800 dark:text-gray-400 dark:group-hover:text-gray-100",
    active && "text-gray-800 dark:text-gray-100",
  );
}

function AdminLayout() {
  const { account, loading } = useAccount();
  const billingProductEnabled = useBillingProductEnabled();
  const links = billingProductEnabled
    ? adminPageLinks
    : adminPageLinks.filter((link) => link.to !== "/admin/polar-webhooks");

  if (loading) {
    return (
      <div className="flex h-full min-h-0 items-center justify-center bg-slate-100 dark:bg-gray-950">
        <div className="flex flex-col items-center space-y-4">
          <div className="h-8 w-8 animate-spin rounded-full border-b border-gray-500 dark:border-gray-400"></div>
          <Text className="text-gray-500 dark:text-gray-400">Loading...</Text>
        </div>
      </div>
    );
  }

  if (!account?.installation_admin) {
    return <Navigate to="/" replace />;
  }

  return (
    <div className="flex h-full min-h-0 flex-col bg-slate-100 dark:bg-gray-950">
      <header className="shrink-0 border-b border-slate-950/15 bg-white dark:border-gray-700/70 dark:bg-gray-900">
        <div className="flex h-12 items-center justify-between px-4">
          <div className="flex items-center gap-3">
            <Link to="/" className="flex items-center">
              <img src={SuperplaneLogo} alt="SuperPlane" className="h-7 w-7 dark:brightness-0 dark:invert" />
            </Link>

            <div className="h-5 w-px bg-slate-200 dark:bg-gray-700" />

            <div className="flex items-center gap-1.5">
              <Shield size={14} className="text-amber-600 dark:text-amber-400" />
              <span className="text-sm font-medium text-gray-800 dark:text-gray-100">Installation Admin</span>
            </div>
          </div>

          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-500 dark:text-gray-400">{account.name}</span>
            <Link
              to="/"
              className="group flex items-center gap-1 text-sm font-medium text-gray-500 transition-colors hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100"
            >
              <ArrowLeft size={14} className="transition-transform group-hover:-translate-x-0.5" />
              Back to app
            </Link>
          </div>
        </div>
      </header>

      <div className="flex min-h-0 flex-1">
        <Sidebar
          className={cn("w-60 shrink-0 border-r bg-white", appDarkModeClasses.sidebarEdge, appDarkModeClasses.surface)}
        >
          <SidebarBody>
            <SidebarSection className="p-4">
              {links.map((link) => (
                <NavLink
                  key={link.to}
                  to={link.to}
                  end={link.end}
                  className={({ isActive }) => adminSidebarNavLinkClass(isActive)}
                >
                  {({ isActive }) => (
                    <>
                      <link.icon size={16} className={adminSidebarNavIconClass(isActive)} />
                      <span className="truncate">{link.label}</span>
                    </>
                  )}
                </NavLink>
              ))}
            </SidebarSection>
          </SidebarBody>
        </Sidebar>

        <main className="min-h-0 min-w-0 flex-1 overflow-auto">
          <div className="mx-auto max-w-7xl px-6 py-6">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}

export default AdminLayout;
