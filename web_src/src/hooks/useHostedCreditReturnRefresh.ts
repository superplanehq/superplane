import { useEffect, useState } from "react";
import {
  HOSTED_CREDIT_REFRESH_INTERVAL_MS,
  HOSTED_CREDIT_REFRESH_TIMEOUT_MS,
  hostedCreditRefreshStatus,
  readHostedCreditGrantSnapshot,
  type HostedCreditRefreshStatus,
} from "@/lib/hostedCredit";

export function useHostedCreditReturnRefresh({
  organizationId,
  creditAdded,
  grantTotalCents,
  refetch,
}: {
  organizationId: string;
  creditAdded: boolean;
  grantTotalCents: number;
  refetch: () => Promise<unknown>;
}): HostedCreditRefreshStatus {
  const [timedOut, setTimedOut] = useState(false);
  const [snapshotCents, setSnapshotCents] = useState<number | null>(null);

  useEffect(() => {
    if (!creditAdded) {
      setTimedOut(false);
      setSnapshotCents(null);
      return;
    }
    setTimedOut(false);
    setSnapshotCents(readHostedCreditGrantSnapshot(organizationId));
  }, [creditAdded, organizationId]);

  const status = hostedCreditRefreshStatus({
    creditAddedQuery: creditAdded,
    snapshotCents,
    grantTotalCents,
    timedOut,
  });

  useEffect(() => {
    if (!creditAdded || status !== "refreshing") {
      return;
    }

    const startedAt = Date.now();
    void refetch();
    const intervalId = window.setInterval(() => {
      void refetch();
      if (Date.now() - startedAt >= HOSTED_CREDIT_REFRESH_TIMEOUT_MS) {
        setTimedOut(true);
      }
    }, HOSTED_CREDIT_REFRESH_INTERVAL_MS);

    return () => window.clearInterval(intervalId);
  }, [creditAdded, refetch, status]);

  return status;
}
