import React, { createContext, useContext, useState, useEffect } from "react";
import { fetchAccount, type Account } from "@/services/authService";

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
    const loadAccount = async () => {
      try {
        const result = await fetchAccount();

        if (result.setupRequired) {
          setSetupRequired(true);
          return;
        }

        if (result.account) {
          setAccount(result.account);
        }
        // If no account, user is not authenticated
      } catch (error) {
        // Network errors or other unexpected errors
        console.error("Failed to fetch account:", error);
      } finally {
        setLoading(false);
      }
    };

    loadAccount();
  }, []);

  return <AccountContext.Provider value={{ account, loading, setupRequired }}>{children}</AccountContext.Provider>;
};
