import { useState } from "react";
import { Toaster } from "sonner";

import {
  ACCOUNT_REDESIGN_PROFILE,
  type AccountRedesignPageId,
  type AccountRedesignProfile,
} from "./accountProfileRedesignMocks";
import { AccountProfileRedesignShell } from "./AccountProfileRedesignShell";
import {
  AccountProfileRedesignProvider,
  AccountProfileRedesignRoutePage,
  AccountSecurityRedesignRoutePage,
} from "./AccountProfileRedesignState";

/**
 * Isolated Account settings playground for unit tests. Storybook factory
 * stories mount the same pages through FactoriesHarness.
 */
export function AccountProfileRedesignPlayground({
  initialPage = "profile",
  initialProfile = ACCOUNT_REDESIGN_PROFILE,
}: {
  initialPage?: AccountRedesignPageId;
  initialProfile?: AccountRedesignProfile;
}) {
  const [page, setPage] = useState<AccountRedesignPageId>(initialPage);
  const [navQuery, setNavQuery] = useState("");

  return (
    <AccountProfileRedesignProvider initialProfile={initialProfile}>
      <AccountProfileRedesignShell
        activePage={page}
        navQuery={navQuery}
        onNavQueryChange={setNavQuery}
        onSelectPage={setPage}
      >
        {page === "profile" ? <AccountProfileRedesignRoutePage /> : null}
        {page === "security" ? <AccountSecurityRedesignRoutePage /> : null}
      </AccountProfileRedesignShell>
      <Toaster position="bottom-center" closeButton />
    </AccountProfileRedesignProvider>
  );
}
