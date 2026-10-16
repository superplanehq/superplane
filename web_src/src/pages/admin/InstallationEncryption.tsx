import type {
  KeysDataKey,
  KeysListKeysResponse,
  KeysListMigrationsResponse,
  KeysMigrationResponse,
} from "@/api-client";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useCallback, useEffect, useState } from "react";

type Action = "rotate" | "migrate" | "rewrap";

const basePath = "/admin/api/installation/keys";
const activeStates = ["pending", "running"];

async function request<T>(path: string, method = "GET", body?: object): Promise<T> {
  const response = await fetch(`${basePath}${path}`, {
    method,
    credentials: "include",
    headers: body ? { "Content-Type": "application/json" } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!response.ok) {
    const detail = await response.text();
    throw new Error(detail || `Request failed (${response.status}).`);
  }
  return response.json() as Promise<T>;
}

function formatDate(value?: string): string {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString();
}

function MigrationStatus({ migration }: { migration: KeysMigrationResponse }) {
  return (
    <div className="mt-4 rounded-md border border-slate-200 p-4 text-sm dark:border-gray-700">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
        <span className="font-medium capitalize text-gray-900 dark:text-gray-100">{migration.state}</span>
        <span className="text-gray-500 dark:text-gray-400">Started {formatDate(migration.createdAt)}</span>
      </div>
      <dl className="mt-3 grid gap-2 text-gray-600 dark:text-gray-400 sm:grid-cols-2">
        <div>
          <dt className="font-medium">Job ID</dt>
          <dd className="break-all font-mono text-xs">{migration.id}</dd>
        </div>
        <div>
          <dt className="font-medium">Target key</dt>
          <dd className="break-all font-mono text-xs">{migration.targetKeyId}</dd>
        </div>
        {migration.state !== "completed" ? (
          <div>
            <dt className="font-medium">Current location</dt>
            <dd>{migration.storageLocation || "Waiting to start"}</dd>
          </div>
        ) : null}
        {migration.state === "completed" ? (
          <div>
            <dt className="font-medium">Completed</dt>
            <dd>{formatDate(migration.completedAt)}</dd>
          </div>
        ) : null}
      </dl>
      {migration.errorMessage ? (
        <Text className="mt-3 break-words text-sm text-red-600 dark:text-red-400">{migration.errorMessage}</Text>
      ) : null}
    </div>
  );
}

type SectionProps = {
  busy: Action | null;
  setConfirm: (action: Action) => void;
};

function DataKeysSection({
  keys,
  activeKeyId,
  unfinished,
  busy,
  setConfirm,
}: SectionProps & {
  keys: KeysDataKey[];
  activeKeyId: string;
  unfinished: boolean;
}) {
  return (
    <section className="py-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className="text-base font-semibold text-gray-900 dark:text-gray-100">Data keys</h2>
          <Text className="mt-1 text-sm text-gray-600 dark:text-gray-400">
            Rotation creates a new key for future values. It does not re-encrypt existing values.
          </Text>
        </div>
        <Button disabled={!!busy || !!unfinished} onClick={() => setConfirm("rotate")}>
          Rotate data key
        </Button>
      </div>
      {unfinished ? (
        <Text className="mt-3 text-sm text-amber-700 dark:text-amber-400">
          Complete the current migration before rotation.
        </Text>
      ) : null}
      <div className="mt-5 overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-slate-200 text-gray-500 dark:border-gray-700 dark:text-gray-400">
            <tr>
              <th className="py-2 font-medium">Key ID</th>
              <th className="py-2 font-medium">Master key</th>
              <th className="py-2 font-medium">Created</th>
              <th className="py-2 font-medium">Status</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-200 dark:divide-gray-700">
            {keys.map((key) => (
              <tr key={key.id}>
                <td className="py-3 pr-4 font-mono text-xs">{key.id}</td>
                <td className="py-3 pr-4 break-all font-mono text-xs">{key.masterKeyRef}</td>
                <td className="py-3 pr-4">{formatDate(key.createdAt)}</td>
                <td className="py-3">{key.id === activeKeyId ? "Active" : "Available"}</td>
              </tr>
            ))}
          </tbody>
        </table>
        {keys.length === 0 ? <Text className="py-4 text-sm text-gray-500">No data keys are available.</Text> : null}
      </div>
    </section>
  );
}

function MigrationSection({
  latestMigration,
  unfinished,
  busy,
  setConfirm,
  refresh,
}: SectionProps & {
  latestMigration?: KeysMigrationResponse;
  unfinished: boolean;
  refresh: () => Promise<void>;
}) {
  return (
    <section className="border-t border-slate-200 py-6 dark:border-gray-700">
      <h2 className="text-base font-semibold text-gray-900 dark:text-gray-100">Re-encrypt stored values</h2>
      <Text className="mt-1 text-sm text-gray-600 dark:text-gray-400">
        Start a migration to convert older values or move stored values to the active data key.
      </Text>
      <div className="mt-4 flex flex-wrap gap-2">
        <Button disabled={!!busy || !!unfinished} onClick={() => setConfirm("migrate")}>
          Start migration
        </Button>
        <Button variant="outline" disabled={!!busy} onClick={() => void refresh()}>
          Refresh status
        </Button>
      </div>
      {latestMigration ? (
        <MigrationStatus migration={latestMigration} />
      ) : (
        <Text className="mt-4 text-sm text-gray-500">No migrations have started.</Text>
      )}
    </section>
  );
}

function MasterKeySection({
  primaryMasterKeyRef,
  busy,
  setConfirm,
}: SectionProps & {
  primaryMasterKeyRef: string;
}) {
  return (
    <section className="border-t border-slate-200 py-6 dark:border-gray-700">
      <h2 className="text-base font-semibold text-gray-900 dark:text-gray-100">Master key</h2>
      <Text className="mt-1 text-sm text-gray-600 dark:text-gray-400">
        Rewrap data keys after you configure a new master key or rotate a KMS key version.
      </Text>
      <div className="mt-4 max-w-xl">
        <Text className="break-all text-sm text-gray-600 dark:text-gray-400">
          Configured target: {primaryMasterKeyRef || "No master key configured"}
        </Text>
        <Button className="mt-4" variant="outline" disabled={!!busy} onClick={() => setConfirm("rewrap")}>
          Rewrap data keys
        </Button>
      </div>
    </section>
  );
}

function ConfirmAction({
  confirm,
  busy,
  setConfirm,
  act,
  primaryMasterKeyRef,
}: {
  confirm: Action | null;
  busy: Action | null;
  setConfirm: (action: Action | null) => void;
  act: (action: Action) => Promise<void>;
  primaryMasterKeyRef: string;
}) {
  const confirmation = {
    rotate: ["Rotate data key", "New values will use a new data key. Existing values will remain readable."],
    migrate: ["Start encryption migration", "This job will re-encrypt stored values with the active data key."],
    rewrap: [
      "Rewrap data keys",
      "The configured master key will protect all data keys. Application values will not change.",
    ],
  } as const;
  return (
    <Dialog open={confirm !== null} onClose={() => (busy ? undefined : setConfirm(null))} size="md">
      {confirm ? (
        <>
          <DialogTitle>{confirmation[confirm][0]}</DialogTitle>
          <DialogDescription className="mt-2 text-sm text-gray-600 dark:text-gray-400">
            {confirmation[confirm][1]}
            {confirm === "rewrap" ? <span className="mt-2 block break-all">Target: {primaryMasterKeyRef}</span> : null}
          </DialogDescription>
          <DialogActions>
            <Button onClick={() => void act(confirm)} disabled={!!busy}>
              {busy ? "Working..." : confirmation[confirm][0]}
            </Button>
            <Button variant="outline" onClick={() => setConfirm(null)} disabled={!!busy}>
              Cancel
            </Button>
          </DialogActions>
        </>
      ) : null}
    </Dialog>
  );
}

export default function InstallationEncryption() {
  const [keys, setKeys] = useState<KeysDataKey[]>([]);
  const [activeKeyId, setActiveKeyId] = useState("");
  const [migrations, setMigrations] = useState<KeysMigrationResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState(false);
  const [busy, setBusy] = useState<Action | null>(null);
  const [confirm, setConfirm] = useState<Action | null>(null);
  const [primaryMasterKeyRef, setPrimaryMasterKeyRef] = useState("");

  const refresh = useCallback(async () => {
    try {
      const [keyData, migrationData] = await Promise.all([
        request<KeysListKeysResponse>(""),
        request<KeysListMigrationsResponse>("/migrations"),
      ]);
      setKeys(keyData.keys ?? []);
      setActiveKeyId(keyData.activeKeyId ?? "");
      setPrimaryMasterKeyRef(keyData.primaryMasterKeyRef ?? "");
      setMigrations(migrationData.migrations ?? []);
      setLoadError(false);
    } catch {
      setLoadError(true);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const latestMigration = migrations[0];
  const unfinished = latestMigration && activeStates.includes(latestMigration.state ?? "");

  useEffect(() => {
    if (latestMigration?.state !== "pending" && latestMigration?.state !== "running") return;
    const timer = window.setInterval(() => void refresh(), 5000);
    return () => window.clearInterval(timer);
  }, [latestMigration?.state, refresh]);

  const act = async (action: Action) => {
    setBusy(action);
    try {
      switch (action) {
        case "rotate":
          await request("/rotate", "POST", {});
          showSuccessToast("New data key is active.");
          break;
        case "migrate":
          await request("/migrations", "POST", {});
          showSuccessToast("Encryption migration started.");
          break;
        case "rewrap":
          await request("/rewrap", "POST", {});
          showSuccessToast("Data keys rewrapped.");
          break;
      }
      setConfirm(null);
      await refresh();
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Could not update encryption keys.");
    } finally {
      setBusy(null);
    }
  };

  return (
    <div>
      <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">Encryption keys</h1>
      <Text className="mt-1 text-sm text-gray-500 dark:text-gray-400">
        Manage data keys for encrypted application values. Keep older keys while stored values or backups need them.
      </Text>

      {loading ? <Text className="mt-6">Loading encryption keys...</Text> : null}
      {loadError ? (
        <div className="mt-6 flex items-center gap-3">
          <Text className="text-sm text-red-600 dark:text-red-400">Could not load encryption keys or migrations.</Text>
          <Button variant="outline" size="sm" onClick={() => void refresh()}>
            Retry
          </Button>
        </div>
      ) : null}

      {!loading && !loadError ? (
        <div className="mt-5 bg-white px-6 dark:bg-gray-900">
          <DataKeysSection
            keys={keys}
            activeKeyId={activeKeyId}
            unfinished={!!unfinished}
            busy={busy}
            setConfirm={setConfirm}
          />
          <MigrationSection
            latestMigration={latestMigration}
            unfinished={!!unfinished}
            busy={busy}
            setConfirm={setConfirm}
            refresh={refresh}
          />
          <MasterKeySection primaryMasterKeyRef={primaryMasterKeyRef} busy={busy} setConfirm={setConfirm} />
        </div>
      ) : null}

      <ConfirmAction
        confirm={confirm}
        busy={busy}
        setConfirm={setConfirm}
        act={act}
        primaryMasterKeyRef={primaryMasterKeyRef}
      />
    </div>
  );
}
