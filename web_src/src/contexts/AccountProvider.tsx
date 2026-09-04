import React, { useCallback, useEffect, useState } from "react";
import { posthog } from "@/posthog";
import { consumePendingSignupAnalyticsPreference } from "@/lib/signupAnalytics";

import { AccountContext, type AccountContextType } from "./accountContextState";

interface AccountProviderProps {
  children: React.ReactNode;
}

export function AccountProvider({ children }: AccountProviderProps) {
  const [account, setAccount] = useState<AccountContextType["account"]>(null);
  const [loading, setLoading] = useState(true);
  const [setupRequired, setSetupRequired] = useState(false);

  const loadAccount = useCallback(async (identify: boolean, keepAccountOnError = false) => {
    const response = await fetch("/account", {
      method: "GET",
      credentials: "include",
      redirect: "manual",
    });

    if (response.status === 409 && response.headers.get("X-Owner-Setup-Required") === "true") {
      setSetupRequired(true);
      setAccount(null);
      return;
    }

    if (response.status !== 200) {
      if (!keepAccountOnError || response.status === 401) {
        setAccount(null);
      }
      return;
    }

    const accountData = await response.json();
    setAccount(accountData);

    if (!identify || accountData.impersonation?.active) {
      return;
    }

    const signupPreference = consumePendingSignupAnalyticsPreference({
      accountEmail: accountData.email,
      currentPath: window.location.pathname,
      signupResult: getSignupAnalyticsResult(window.location.search),
    });

    const accountProperties = {
      email: accountData.email,
      name: accountData.name,
      installation_admin: accountData.installation_admin,
      ...(signupPreference
        ? {
            product_updates_opt_in: signupPreference.productUpdatesOptIn,
          }
        : {}),
    };

    posthog.identify(accountData.id, accountProperties);

    if (signupPreference) {
      posthog.capture("auth:signup", {
        product_updates_opt_in: signupPreference.productUpdatesOptIn,
        $set: {
          product_updates_opt_in: signupPreference.productUpdatesOptIn,
        },
      });
    }
  }, []);

  useEffect(() => {
    loadAccount(true)
      .catch(() => undefined)
      .finally(() => setLoading(false));
  }, [loadAccount]);

  const refreshAccount = useCallback(async () => {
    await loadAccount(false, true);
  }, [loadAccount]);

  return (
    <AccountContext.Provider value={{ account, loading, setupRequired, refreshAccount }}>
      {children}
    </AccountContext.Provider>
  );
}

function getSignupAnalyticsResult(search: string) {
  const value = new URLSearchParams(search).get("auth_signup_result");
  if (value === "created" || value === "existing") {
    return value;
  }

  return null;
}
