import SuperplaneLogo from "@/assets/superplane.svg";
import { Text } from "@/components/Text/text";
import { useAccount } from "@/contexts/useAccount";
import { ArrowLeft, Building, Network, Shield, Terminal, Users } from "lucide-react";
import React from "react";
import { Link, Navigate, NavLink, Outlet } from "react-router-dom";

const AdminLayout: React.FC = () => {
  const { account, loading } = useAccount();

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-surface-canvas">
        <div className="flex flex-col items-center space-y-4">
          <div className="h-8 w-8 animate-spin rounded-full border-b border-focus-ring"></div>
          <Text className="text-content-secondary">Loading...</Text>
        </div>
      </div>
    );
  }

  if (!account?.installation_admin) {
    return <Navigate to="/" replace />;
  }

  const navLinkClass = ({ isActive }: { isActive: boolean }) =>
    `flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded transition-colors ${
      isActive
        ? "bg-surface-subtle text-content-primary"
        : "text-content-secondary hover:bg-surface-subtle hover:text-content-primary"
    }`;

  return (
    <div className="flex min-h-screen flex-col bg-surface-canvas">
      <header className="border-b border-edge-default bg-surface-default">
        <div className="px-4 h-12 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Link to="/" className="flex items-center">
              <img src={SuperplaneLogo} alt="SuperPlane" className="w-7 h-7 dark:brightness-0 dark:invert" />
            </Link>

            <div className="h-5 w-px bg-edge-default" />

            <div className="flex items-center gap-1.5">
              <Shield size={14} className="text-amber-600 dark:text-amber-400" />
              <span className="text-sm font-medium text-content-primary">Installation Admin</span>
            </div>

            <div className="h-5 w-px bg-edge-default" />

            <nav className="flex items-center gap-1">
              <NavLink to="/admin" end className={navLinkClass}>
                <Building size={14} />
                Organizations
              </NavLink>
              <NavLink to="/admin/accounts" className={navLinkClass}>
                <Users size={14} />
                Accounts
              </NavLink>
              <NavLink to="/admin/settings" className={navLinkClass}>
                <Network size={14} />
                Settings
              </NavLink>
              <NavLink to="/admin/runner-tasks" className={navLinkClass}>
                <Terminal size={14} />
                Runner Tasks
              </NavLink>
            </nav>
          </div>

          <div className="flex items-center gap-4">
            <span className="text-sm text-content-secondary">{account.name}</span>
            <Link
              to="/"
              className="group flex items-center gap-1 text-sm font-medium text-content-secondary transition-colors hover:text-content-primary"
            >
              <ArrowLeft size={14} className="transition-transform group-hover:-translate-x-0.5" />
              Back to app
            </Link>
          </div>
        </div>
      </header>

      <main className="w-full flex-1">
        <div className="max-w-7xl mx-auto px-6 py-6">
          <Outlet />
        </div>
      </main>
    </div>
  );
};

export default AdminLayout;
