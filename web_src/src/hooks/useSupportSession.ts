import { useAccount } from "@/contexts/useAccount";

/** True while an installation admin views the app in a read-only support session. */
export function useIsSupportSession(): boolean {
  const { account } = useAccount();
  return account?.support_session?.active === true;
}
