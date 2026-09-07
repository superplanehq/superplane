import type { AccountContextType } from "@/contexts/accountContextState";

type Account = NonNullable<AccountContextType["account"]>;

/** Seeds the new organization before GitHub connect. Do not use a GitHub identity here. */
export function organizationNameFromAccount(account: Account | null | undefined): string {
  const name = account?.name?.trim();
  if (name) {
    return name;
  }

  const email = account?.email?.trim() ?? "";
  const localPart = email.split("@")[0]?.trim() ?? "";
  return localPart;
}
