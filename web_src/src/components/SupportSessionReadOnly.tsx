import type { ReactNode } from "react";
import { useIsSupportSession } from "@/hooks/useSupportSession";
import { SUPPORT_SESSION_READ_ONLY_MESSAGE } from "@/lib/supportSession";

/**
 * Disables every native control inside `children` during a read-only support
 * session. Use it for pages whose actions have no RBAC check, such as account
 * settings.
 */
export function SupportSessionReadOnly({ children }: { children: ReactNode }) {
  const inSupportSession = useIsSupportSession();
  if (!inSupportSession) {
    return <>{children}</>;
  }

  return (
    <fieldset disabled className="contents" data-testid="support-session-read-only">
      <p
        role="status"
        className="mb-4 rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-[13px] text-amber-900 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-200"
      >
        {SUPPORT_SESSION_READ_ONLY_MESSAGE}
      </p>
      {children}
    </fieldset>
  );
}
