import { useCallback, useEffect, useState } from "react";

import {
  adminSsoDisableOrganizationSso,
  adminSsoGetOrganizationSso,
  adminSsoVerifyOrganizationDomain,
} from "@/api-client/sdk.gen";
import type { SsoAdminConnection, SsoAdminDomain } from "@/api-client/types.gen";
import { Button } from "@/components/ui/button";
import { getApiErrorMessage } from "@/lib/errors";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

export function OrgSsoSection({
  orgId,
  preview,
}: {
  orgId: string;
  preview?: { connection: SsoAdminConnection | null; domains: SsoAdminDomain[]; error?: string | null };
}) {
  const [connection, setConnection] = useState<SsoAdminConnection | null>(preview?.connection ?? null);
  const [domains, setDomains] = useState<SsoAdminDomain[]>(preview?.domains ?? []);
  const [error, setError] = useState<string | null>(preview?.error ?? null);
  const [pending, setPending] = useState(false);
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
    if (preview) {
      return;
    }
    load();
  }, [load, preview]);

  return (
    <section className="mt-6 space-y-3" data-testid="admin-sso">
      <h3 className="text-sm font-medium">Single Sign-On</h3>
      {error ? <p role="alert">{error}</p> : null}
      <p>Status: {connectionStatus(connection)}</p>
      <p>Enforcement: {connection?.enforcement || "none"}</p>
      <AdminSsoDomains orgId={orgId} domains={domains} load={load} setConfirm={setConfirm} />
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
      <AdminSsoConfirm
        confirm={confirm}
        pending={pending}
        setPending={setPending}
        setConfirm={setConfirm}
        setError={setError}
      />
    </section>
  );
}

function connectionStatus(connection: SsoAdminConnection | null) {
  if (!connection?.exists) {
    return "Not connected";
  }
  return connection.status || "unknown";
}

function AdminSsoDomains({
  orgId,
  domains,
  load,
  setConfirm,
}: {
  orgId: string;
  domains: SsoAdminDomain[];
  load: () => void;
  setConfirm: (value: { title: string; body: string; run: () => Promise<void> }) => void;
}) {
  return (
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
  );
}

function AdminSsoConfirm({
  confirm,
  pending,
  setPending,
  setConfirm,
  setError,
}: {
  confirm: { title: string; body: string; run: () => Promise<void> } | null;
  pending: boolean;
  setPending: (value: boolean) => void;
  setConfirm: (value: null) => void;
  setError: (value: string) => void;
}) {
  return (
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
            disabled={pending}
            onClick={() => runAdminConfirm(confirm, setPending, setConfirm, setError)}
          >
            Confirm
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function runAdminConfirm(
  confirm: { run: () => Promise<void> } | null,
  setPending: (value: boolean) => void,
  setConfirm: (value: null) => void,
  setError: (value: string) => void,
) {
  if (!confirm) {
    return;
  }
  setPending(true);
  void confirm
    .run()
    .then(() => setConfirm(null))
    .catch((cause: unknown) => {
      setError(getApiErrorMessage(cause, "The request failed. Try again."));
      setConfirm(null);
    })
    .finally(() => setPending(false));
}
