import React, { useEffect } from "react";
import { useAccount } from "../contexts/AccountContext";
import { useNavigate, useLocation } from "react-router-dom";

interface AuthGuardProps {
  children: React.ReactNode;
}

const AuthGuard: React.FC<AuthGuardProps> = ({ children }) => {
  const { account, loading } = useAccount();
  const navigate = useNavigate();
  const location = useLocation();

  const shouldRedirectToLogin = !loading && !account;

  useEffect(() => {
    if (!shouldRedirectToLogin) {
      return;
    }

    console.log("[AuthGuard] No account, redirecting to login from:", location.pathname);
    const redirectParam = encodeURIComponent(`${location.pathname}${location.search}`);
    navigate(`/login?redirect=${redirectParam}`, { replace: true });
  }, [location.pathname, location.search, navigate, shouldRedirectToLogin]);

  // Show loading spinner while fetching account
  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b border-blue-600"></div>
          <p className="text-sm text-gray-500 dark:text-gray-400">Loading...</p>
        </div>
      </div>
    );
  }

  if (shouldRedirectToLogin) {
    return null;
  }

  // Account is authenticated, render the protected content
  return <>{children}</>;
};

export default AuthGuard;
