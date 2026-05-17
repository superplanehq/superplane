import React, { useEffect, useState } from "react";
import { posthog } from "@/posthog";

import { AccountContext, type AccountContextType } from "./accountContext";

interface AccountProviderProps {
  children: React.ReactNode;
}

export const AccountProvider: React.FC<AccountProviderProps> = ({ children }) => {
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
            posthog.identify(accountData.id, {
              email: accountData.email,
              name: accountData.name,
              installation_admin: accountData.installation_admin,
            });
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
};
