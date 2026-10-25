import { useEffect, useState, type ReactNode } from "react";
import { Navigate, useLocation } from "react-router";

import { useAccount } from "@/contexts/useAccount";

const SAFE_SETUP_STEP = /^[a-z0-9-]+$/;

function isOwnerSetupPath(pathname: string) {
  return pathname === "/setup" || pathname.startsWith("/setup/");
}

function openOwnerSetupPath(step: string | undefined) {
  if (!step || !SAFE_SETUP_STEP.test(step)) {
    return "/setup";
  }
  return `/setup?step=${step}`;
}

function useOpenOwnerSetupPath(accountLoading: boolean, installationAdmin: boolean) {
  const [path, setPath] = useState<string | null>(null);
  const [resolved, setResolved] = useState(false);
  const shouldCheck = !accountLoading && installationAdmin;

  useEffect(() => {
    if (!shouldCheck) {
      return;
    }

    let cancelled = false;
    setResolved(false);
    setPath(null);

    void fetch("/api/v1/setup-owner/status", { credentials: "include" })
      .then(async (response) => {
        if (!response.ok) {
          return null;
        }
        return (await response.json()) as { status?: string; step?: string };
      })
      .then((body) => {
        if (cancelled) {
          return;
        }
        setPath(body?.status === "in_progress" ? openOwnerSetupPath(body.step) : null);
        setResolved(true);
      })
      .catch(() => {
        if (!cancelled) {
          setPath(null);
          setResolved(true);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [shouldCheck]);

  if (!shouldCheck) {
    return { loading: false, path: null };
  }

  return { loading: !resolved, path };
}

export function SetupGuard({ children }: { children: ReactNode }) {
  const { account, loading, setupRequired } = useAccount();
  const location = useLocation();
  const openSetup = useOpenOwnerSetupPath(loading, account?.installation_admin === true);

  if (isOwnerSetupPath(location.pathname)) {
    return children;
  }

  if (!loading && setupRequired) {
    return <Navigate to="/setup" replace />;
  }

  if (openSetup.loading) {
    return null;
  }

  if (openSetup.path) {
    return <Navigate to={openSetup.path} replace />;
  }

  return children;
}
