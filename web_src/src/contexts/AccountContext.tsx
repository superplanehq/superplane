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
          redirect: "manual", // Don't follow redirects, check status code instead
        });

        if (response.status === 409 && response.headers.get("X-Owner-Setup-Required") === "true") {
          setSetupRequired(true);
          return;
        }

        if (response.status === 200) {
          const accountData = await response.json();
          setAccount(accountData);
        }
        // If response is not 200 (e.g., 307 redirect, 401, etc.), user is not authenticated
      } catch (error) {
        // Network errors or other unexpected errors
        console.error("Failed to fetch account:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchAccount();
  }, []);

  return <AccountContext.Provider value={{ account, loading, setupRequired }}>{children}</AccountContext.Provider>;
};
