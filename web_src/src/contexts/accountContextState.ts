import { createContext } from "react";
import type { AccountLicense } from "@/lib/license";

interface AccountSupportSession {
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
  provider_id: string;
  username: string;
  name?: string;
  avatar_url?: string;
  active: boolean;
}

/**
 * Whether factory pull requests open with the member's GitHub account instead
 * of the SuperPlane GitHub App.
 */
export interface AccountGitHubPrAuthorization {
  available: boolean;
  enabled: boolean;
  login?: string;
  needs_reauthorization: boolean;
}

export interface AccountOrganizationPendingDeletion {
  id: string;
  name: string;
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
  github_pr_authorization?: AccountGitHubPrAuthorization;
  organizations_pending_deletion?: AccountOrganizationPendingDeletion[];
  roles?: string[];
  groups?: string[];
  support_session?: AccountSupportSession;
  license?: AccountLicense;
  /** Product billing. False hides Billing. Missing keeps Billing. */
  billing_enabled?: boolean;
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
