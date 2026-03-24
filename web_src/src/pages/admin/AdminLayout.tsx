import SuperplaneLogo from "@/assets/superplane.svg";
import { Text } from "@/components/Text/text";
import { useAccount } from "@/contexts/AccountContext";
import { ArrowLeft, Building, Shield, Users } from "lucide-react";
import React from "react";
import { Link, Navigate, NavLink, Outlet } from "react-router-dom";

const AdminLayout: React.FC = () => {
  const { account, loading } = useAccount();

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-100">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b border-gray-500"></div>
          <Text className="text-gray-500">Loading...</Text>
        </div>
      </div>
    );
  }

  if (!account?.installation_admin) {
    return <Navigate to="/" replace />;
  }

  const navLinkClass = ({ isActive }: { isActive: boolean }) =>
    `flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded transition-colors ${
      isActive ? "bg-slate-200/80 text-gray-800" : "text-gray-500 hover:text-gray-800 hover:bg-slate-100"
    }`;

  return (
    <div className="min-h-screen flex flex-col bg-slate-100">
      <header className="bg-white border-b border-slate-950/15">
        <div className="px-4 h-12 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Link to="/" className="flex items-center">
              <img src={SuperplaneLogo} alt="SuperPlane" className="w-7 h-7" />
            </Link>

            <div className="h-5 w-px bg-slate-200" />

            <div className="flex items-center gap-1.5">
              <Shield size={14} className="text-amber-600" />
              <span className="text-sm font-medium text-gray-800">Installation Admin</span>
            </div>

            <div className="h-5 w-px bg-slate-200" />

            <nav className="flex items-center gap-1">
              <NavLink to="/admin" end className={navLinkClass}>
                <Building size={14} />
                Organizations
              </NavLink>
              <NavLink to="/admin/accounts" className={navLinkClass}>
                <Users size={14} />
                Accounts
              </NavLink>
            </nav>
          </div>

          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-500">{account.name}</span>
            <Link
              to="/"
              className="group flex items-center gap-1 text-sm font-medium text-gray-500 hover:text-gray-800 transition-colors"
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
