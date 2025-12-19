import React from "react";
import { useAccount } from "../contexts/AccountContext";
import { useNavigate, useLocation } from "react-router-dom";

interface AuthGuardProps {
  children: React.ReactNode;
}

const AuthGuard: React.FC<AuthGuardProps> = ({ children }) => {
  const { account, loading } = useAccount();
  const navigate = useNavigate();
  const location = useLocation();

  // If account is not loaded and not loading, redirect to organization select
  if (!loading && !account) {
    console.log("[AuthGuard] No account, redirecting to organization select from:", location.pathname);
    navigate("/", { replace: true });
    return null;
  }

  // Show loading spinner while fetching account
  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b border-blue-600"></div>
          <p className="text-sm text-gray-600 dark:text-gray-400">Loading...</p>
        </div>
      </div>
    );
  }

  // Account is authenticated, render the protected content
  return <>{children}</>;
};

export default AuthGuard;
