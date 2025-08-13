import React, { createContext, useContext, useState, useEffect } from 'react';

interface Account {
  id: string;
  name: string;
  email: string;
}

interface AccountContextType {
  account: Account | null;
  loading: boolean;
}

const AccountContext = createContext<AccountContextType>({
  account: null,
  loading: true,
});

export const useAccount = () => {
  const context = useContext(AccountContext);
  if (!context) {
    throw new Error('useAccount must be used within an AccountProvider');
  }
  return context;
};

interface AccountProviderProps {
  children: React.ReactNode;
}

export const AccountProvider: React.FC<AccountProviderProps> = ({ children }) => {
  const [account, setAccount] = useState<Account | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchAccount = async () => {
      try {
        const response = await fetch('/account', {
          method: 'GET',
          credentials: 'include',
        });
        
        if (response.ok) {
          const accountData = await response.json();
          setAccount(accountData);
        }
      } catch (error) {
        console.error('Failed to fetch account:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchAccount();
  }, []);

  return (
    <AccountContext.Provider value={{ account, loading }}>
      {children}
    </AccountContext.Provider>
  );
};