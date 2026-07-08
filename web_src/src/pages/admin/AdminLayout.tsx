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
      <div className="min-h-screen flex items-center justify-center bg-slate-100 dark:bg-gray-950">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b border-gray-500 dark:border-gray-400"></div>
          <Text className="text-gray-500 dark:text-gray-400">Loading...</Text>
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
        ? "bg-slate-200/80 text-gray-800 dark:bg-gray-800 dark:text-gray-100"
        : "text-gray-500 hover:text-gray-800 hover:bg-slate-100 dark:text-gray-400 dark:hover:text-gray-100 dark:hover:bg-gray-800"
    }`;

  return (
    <div className="min-h-screen flex flex-col bg-slate-100 dark:bg-gray-950">
      <header className="bg-white border-b border-slate-950/15 dark:bg-gray-900 dark:border-gray-700/70">
        <div className="px-4 h-12 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Link to="/" className="flex items-center">
              <img src={SuperplaneLogo} alt="SuperPlane" className="w-7 h-7 dark:brightness-0 dark:invert" />
            </Link>

            <div className="h-5 w-px bg-slate-200 dark:bg-gray-700" />

            <div className="flex items-center gap-1.5">
              <Shield size={14} className="text-amber-600 dark:text-amber-400" />
              <span className="text-sm font-medium text-gray-800 dark:text-gray-100">Installation Admin</span>
            </div>

            <div className="h-5 w-px bg-slate-200 dark:bg-gray-700" />

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
            <span className="text-sm text-gray-500 dark:text-gray-400">{account.name}</span>
            <Link
              to="/"
              className="group flex items-center gap-1 text-sm font-medium text-gray-500 hover:text-gray-800 transition-colors dark:text-gray-400 dark:hover:text-gray-100"
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
