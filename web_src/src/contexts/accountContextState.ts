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

/**
 * An identity the account owns on another service. A linked account is not a
 * sign-in method: it tells SuperPlane which activity on that service belongs to
 * this account.
 */
export interface AccountLinkedAccount {
  provider: string;
  username: string;
  name?: string;
  avatar_url?: string;
}

interface Account {
  id: string;
  name: string;
  email: string;
  avatar_url: string;
  installation_admin: boolean;
  has_password: boolean;
  providers?: AccountProviderMethod[];
  linked_accounts?: AccountLinkedAccount[];
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
