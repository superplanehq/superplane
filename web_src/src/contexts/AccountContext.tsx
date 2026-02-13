import React, { createContext, useContext, useState, useEffect } from "react";

interface Account {
  id: string;
  name: string;
  email: string;
  avatar_url: string;
}

interface AccountContextType {
  account: Account | null;
  loading: boolean;
  setupRequired: boolean;
}

const AccountContext = createContext<AccountContextType>({
  account: null,
  loading: true,
  setupRequired: false,
});

export const useAccount = () => {
  const context = useContext(AccountContext);
  if (!context) {
    throw new Error("useAccount must be used within an AccountProvider");
  }
  return context;
};

interface AccountProviderProps {
  children: React.ReactNode;
}

export const AccountProvider: React.FC<AccountProviderProps> = ({ children }) => {
  const [account, setAccount] = useState<Account | null>(null);
  const [loading, setLoading] = useState(true);
  const [setupRequired, setSetupRequired] = useState(false);

  useEffect(() => {
    const fetchAccount = async () => {
      try {
        const response = await fetch("/account", {
          method: "GET",
          credentials: "include",
        });

        if (response.status === 409 && response.headers.get("X-Owner-Setup-Required") === "true") {
          setSetupRequired(true);
          setAccount(null);
          return;
        }

        if (!response.ok) {
          setAccount(null);
          setSetupRequired(false);
          return;
        }

        const contentType = response.headers.get("Content-Type") ?? "";
        if (!contentType.includes("application/json")) {
          setAccount(null);
          setSetupRequired(false);
          return;
        }

        const accountData = (await response.json()) as Account;
        setAccount(accountData);
        setSetupRequired(false);
      } catch (_error) {
        // Network errors or other unexpected errors
        setAccount(null);
      } finally {
        setLoading(false);
      }
    };

    fetchAccount();
  }, []);

  return <AccountContext.Provider value={{ account, loading, setupRequired }}>{children}</AccountContext.Provider>;
};
