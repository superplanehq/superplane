import React, { useEffect, useState } from "react";
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

  useEffect(() => {
    const fetchAccount = async () => {
      try {
        const response = await fetch("/account", {
          method: "GET",
          credentials: "include",
          redirect: "manual", // Don't follow redirects, check status code instead
        });

        if (response.status === 409 && response.headers.get("X-Owner-Setup-Required") === "true") {
          setSetupRequired(true);
          return;
        }

        if (response.status === 200) {
          const accountData = await response.json();
          setAccount(accountData);

          if (!accountData.impersonation?.active) {
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
          }
        }
        // If response is not 200 (e.g., 307 redirect, 401, etc.), user is not authenticated
      } catch {
        // Network errors or other unexpected errors
      } finally {
        setLoading(false);
      }
    };

    fetchAccount();
  }, []);

  return <AccountContext.Provider value={{ account, loading, setupRequired }}>{children}</AccountContext.Provider>;
}

function getSignupAnalyticsResult(search: string) {
  const value = new URLSearchParams(search).get("auth_signup_result");
  if (value === "created" || value === "existing") {
    return value;
  }

  return null;
}
