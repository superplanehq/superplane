import { useEffect, useState } from "react";
import {
  HOSTED_CREDIT_REFRESH_INTERVAL_MS,
  HOSTED_CREDIT_REFRESH_TIMEOUT_MS,
  hostedCreditRefreshStatus,
  readHostedCreditGrantSnapshot,
  type HostedCreditRefreshStatus,
} from "@/lib/hostedCredit";

export function useHostedCreditReturnRefresh(args: {
  organizationId: string;
  creditAdded: boolean;
  grantTotalCents: number;
  refetch: () => Promise<unknown>;
}): HostedCreditRefreshStatus {
  const [timedOut, setTimedOut] = useState(false);
  const [snapshotCents, setSnapshotCents] = useState<number | null>(null);

  useEffect(() => {
    if (!args.creditAdded) {
      setTimedOut(false);
      setSnapshotCents(null);
      return;
    }
    setTimedOut(false);
    setSnapshotCents(readHostedCreditGrantSnapshot(args.organizationId));
  }, [args.creditAdded, args.organizationId]);

  const status = hostedCreditRefreshStatus({
    creditAddedQuery: args.creditAdded,
    snapshotCents,
    grantTotalCents: args.grantTotalCents,
    timedOut,
  });

  useEffect(() => {
    if (!args.creditAdded || status !== "refreshing") {
      return;
    }

    const startedAt = Date.now();
    void args.refetch();
    const intervalId = window.setInterval(() => {
      void args.refetch();
      if (Date.now() - startedAt >= HOSTED_CREDIT_REFRESH_TIMEOUT_MS) {
        setTimedOut(true);
      }
    }, HOSTED_CREDIT_REFRESH_INTERVAL_MS);

    return () => window.clearInterval(intervalId);
  }, [args.creditAdded, args.refetch, status]);

  return status;
}
