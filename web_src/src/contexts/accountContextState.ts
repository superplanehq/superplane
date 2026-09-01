import { createContext } from "react";

interface AccountImpersonation {
  active: boolean;
  user_name?: string;
}

export interface ConnectedAccountProvider {
  provider: string;
  username: string;
  display_name: string;
  email: string;
  avatar_url: string;
}

interface Account {
  id: string;
  name: string;
  email: string;
  avatar_url: string;
  installation_admin: boolean;
  has_password: boolean;
  providers?: ConnectedAccountProvider[];
  roles?: string[];
  groups?: string[];
  impersonation?: AccountImpersonation;
}

export interface AccountContextType {
  account: Account | null;
  loading: boolean;
  setupRequired: boolean;
}

export const AccountContext = createContext<AccountContextType>({
  account: null,
  loading: true,
  setupRequired: false,
});
