import { createContext } from "react";

interface AccountImpersonation {
  active: boolean;
  user_name?: string;
}

export interface AccountProviderMethod {
  provider: string;
  email?: string;
  username?: string;
}

interface Account {
  id: string;
  name: string;
  email: string;
  avatar_url: string;
  installation_admin: boolean;
  has_password: boolean;
  providers?: AccountProviderMethod[];
  roles?: string[];
  groups?: string[];
  impersonation?: AccountImpersonation;
}

export interface AccountContextType {
  account: Account | null;
  loading: boolean;
  setupRequired: boolean;
  refreshAccount: () => Promise<void>;
}

export const AccountContext = createContext<AccountContextType>({
  account: null,
  loading: true,
  setupRequired: false,
  refreshAccount: async () => undefined,
});
