import { useCallback, useEffect, useState } from "react";

import {
  adminSsoDisableOrganizationSso,
  adminSsoGetOrganizationSso,
  adminSsoVerifyOrganizationDomain,
} from "@/api-client/sdk.gen";
import type { SsoAdminConnection, SsoAdminDomain } from "@/api-client/types.gen";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

export function OrgSsoSection({ orgId }: { orgId: string }) {
  const [connection, setConnection] = useState<SsoAdminConnection | null>(null);
  const [domains, setDomains] = useState<SsoAdminDomain[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [confirm, setConfirm] = useState<null | { title: string; body: string; run: () => Promise<void> }>(null);

  const load = useCallback(() => {
    adminSsoGetOrganizationSso({ path: { organizationId: orgId } })
      .then((response) => {
        setConnection(response.data?.connection ?? null);
        setDomains(response.data?.domains ?? []);
        setError(null);
      })
      .catch(() => setError("Single Sign-On details are not available."));
  }, [orgId]);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <section className="mt-6 space-y-3" data-testid="admin-sso">
      <h3 className="text-sm font-medium">Single Sign-On</h3>
      {error ? <p role="alert">{error}</p> : null}
      <p>Status: {connection?.exists ? connection.status || "unknown" : "Not connected"}</p>
      <p>Enforcement: {connection?.enforcement || "none"}</p>
      <ul>
        {domains.map((domain) => (
          <li key={domain.id} className="flex items-center gap-2">
            <span>{domain.domain}</span>
            <span>{domain.verified ? "Verified" : "Pending"}</span>
            {!domain.verified && domain.id ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() =>
                  setConfirm({
                    title: "Verify domain manually",
                    body: "Mark this domain as verified without a DNS lookup.",
                    run: async () => {
                      await adminSsoVerifyOrganizationDomain({
                        path: { organizationId: orgId, domainId: domain.id! },
                      });
                      load();
                    },
                  })
                }
              >
                Verify domain manually
              </Button>
            ) : null}
          </li>
        ))}
      </ul>
      <Button
        type="button"
        variant="outline"
        disabled={!connection?.exists}
        onClick={() =>
          setConfirm({
            title: "Disable connection",
            body: "Members can sign in with other methods again.",
            run: async () => {
              await adminSsoDisableOrganizationSso({ path: { organizationId: orgId } });
              load();
            },
          })
        }
      >
        Disable connection
      </Button>
      <Dialog open={Boolean(confirm)} onOpenChange={(open) => !open && setConfirm(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{confirm?.title}</DialogTitle>
            <DialogDescription>{confirm?.body}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setConfirm(null)}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={() => {
                void confirm?.run().finally(() => setConfirm(null));
              }}
            >
              Confirm
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  );
}
