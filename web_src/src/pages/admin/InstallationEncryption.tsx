import type {
  KeysDataKey,
  KeysListKeysResponse,
  KeysListMigrationsResponse,
  KeysMigrationResponse,
} from "@/api-client";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useCallback, useEffect, useState } from "react";

type Action = "create" | "migrate" | "rewrap";
type MigrationError = "unavailable" | "load" | null;

const basePath = "/admin/api/installation/keys";
const activeStates = ["pending", "running", "auditing"];

class RequestError extends Error {
  constructor(
    message: string,
    readonly status: number,
  ) {
    super(message);
  }
}

async function request<T>(path: string, method = "GET", body?: object): Promise<T> {
  const response = await fetch(`${basePath}${path}`, {
    method,
    credentials: "include",
    headers: body ? { "Content-Type": "application/json" } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!response.ok) {
    const detail = await response.text();
    throw new RequestError(detail || `Request failed (${response.status}).`, response.status);
  }
  return response.json() as Promise<T>;
}

function formatDate(value?: string): string {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString();
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
            A new data key protects future values. Existing values keep their current keys.
          </Text>
        </div>
        <Button disabled={!!busy || !!unfinished} onClick={() => setConfirm("create")}>
          Create data key
        </Button>
      </div>
      {unfinished ? (
        <Text className="mt-3 text-sm text-amber-700 dark:text-amber-400">
          Complete the current migration before you create a data key.
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
  migrations,
  unfinished,
  busy,
  setConfirm,
  migrationError,
  onRetry,
}: SectionProps & {
  migrations: KeysMigrationResponse[];
  unfinished: boolean;
  migrationError: MigrationError;
  onRetry: () => void;
}) {
  const showMigrations = migrationError === null || (migrationError === "load" && migrations.length > 0);
  return (
    <section className="border-t border-slate-200 py-6 dark:border-gray-700">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className="text-base font-semibold text-gray-900 dark:text-gray-100">Re-encrypt stored values</h2>
          <Text className="mt-1 text-sm text-gray-600 dark:text-gray-400">
            Start a migration to convert older values or move stored values to the active data key.
          </Text>
        </div>
        <Button disabled={!!busy || !!unfinished || migrationError !== null} onClick={() => setConfirm("migrate")}>
          Start migration
        </Button>
      </div>
      {migrationError ? (
        <div className="mt-5 flex items-center gap-3">
          <Text className="text-sm text-red-600 dark:text-red-400" role="status">
            {migrationError === "unavailable"
              ? "Migrations are unavailable while legacy writes are enabled."
              : "Could not load migrations."}
          </Text>
          {migrationError === "load" ? (
            <Button variant="outline" size="sm" onClick={onRetry}>
              Retry
            </Button>
          ) : null}
        </div>
      ) : null}
      {showMigrations ? (
        <div className="mt-5 overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-slate-200 text-gray-500 dark:border-gray-700 dark:text-gray-400">
              <tr>
                <th className="py-2 font-medium">Job ID</th>
                <th className="py-2 font-medium">Target key</th>
                <th className="py-2 font-medium">Started</th>
                <th className="py-2 font-medium">Status</th>
                <th className="py-2 font-medium">Current location</th>
                <th className="py-2 font-medium">Completed</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 dark:divide-gray-700">
              {migrations.map((migration) => (
                <tr key={migration.id}>
                  <td className="py-3 pr-4 break-all font-mono text-xs">{migration.id}</td>
                  <td className="py-3 pr-4 break-all font-mono text-xs">{migration.targetKeyId}</td>
                  <td className="py-3 pr-4 whitespace-nowrap">{formatDate(migration.createdAt)}</td>
                  <td className="py-3 pr-4">
                    <span className="font-medium capitalize">{migration.state}</span>
                    {migration.errorMessage ? (
                      <Text className="mt-1 break-words text-sm text-red-600 dark:text-red-400">
                        {migration.errorMessage}
                      </Text>
                    ) : null}
                  </td>
                  <td className="py-3 pr-4">
                    {migration.state === "completed" ? "—" : migration.storageLocation || "Waiting to start"}
                  </td>
                  <td className="py-3 whitespace-nowrap">{formatDate(migration.completedAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {migrations.length === 0 ? (
            <Text className="py-4 text-sm text-gray-500">No migrations have started.</Text>
          ) : null}
        </div>
      ) : null}
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
    create: ["Create data key", "New values will use the new data key. Existing values will remain readable."],
    migrate: ["Start encryption migration", "This job will re-encrypt stored values with the active data key."],
    rewrap: [
      "Rewrap data keys",
      "The configured master key will protect all data keys. Application values will not change.",
    ],
  } as const;
  return (
    <Dialog open={confirm !== null} onOpenChange={(open) => !open && !busy && setConfirm(null)}>
      {confirm ? (
        <DialogContent showCloseButton={!busy}>
          <DialogHeader>
            <DialogTitle>{confirmation[confirm][0]}</DialogTitle>
            <DialogDescription>
              {confirmation[confirm][1]}
              {confirm === "rewrap" ? (
                <span className="mt-2 block break-all">Target: {primaryMasterKeyRef}</span>
              ) : null}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirm(null)} disabled={!!busy}>
              Cancel
            </Button>
            <Button onClick={() => void act(confirm)} disabled={!!busy}>
              {busy ? "Working..." : confirmation[confirm][0]}
            </Button>
          </DialogFooter>
        </DialogContent>
      ) : null}
    </Dialog>
  );
}

export function InstallationEncryption() {
  const [keys, setKeys] = useState<KeysDataKey[]>([]);
  const [activeKeyId, setActiveKeyId] = useState("");
  const [migrations, setMigrations] = useState<KeysMigrationResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState(false);
  const [migrationError, setMigrationError] = useState<MigrationError>(null);
  const [busy, setBusy] = useState<Action | null>(null);
  const [confirm, setConfirm] = useState<Action | null>(null);
  const [primaryMasterKeyRef, setPrimaryMasterKeyRef] = useState("");

  const refresh = useCallback(async () => {
    const [keyResult, migrationResult] = await Promise.allSettled([
      request<KeysListKeysResponse>(""),
      request<KeysListMigrationsResponse>("/migrations"),
    ]);

    if (keyResult.status === "rejected") {
      setLoadError(true);
    } else {
      setKeys(keyResult.value.keys ?? []);
      setActiveKeyId(keyResult.value.activeKeyId ?? "");
      setPrimaryMasterKeyRef(keyResult.value.primaryMasterKeyRef ?? "");
      setLoadError(false);
    }
    if (migrationResult.status === "rejected") {
      if (migrationResult.reason instanceof RequestError && migrationResult.reason.status === 400) {
        setMigrations([]);
        setMigrationError("unavailable");
      } else {
        setMigrationError("load");
      }
    } else {
      setMigrations(migrationResult.value.migrations ?? []);
      setMigrationError(null);
    }
    setLoading(false);
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const latestMigration = migrations[0];
  const unfinished = latestMigration && activeStates.includes(latestMigration.state ?? "");

  useEffect(() => {
    if (!activeStates.includes(latestMigration?.state ?? "")) return;
    const timer = window.setInterval(() => void refresh(), 5000);
    return () => window.clearInterval(timer);
  }, [latestMigration?.state, refresh]);

  const act = async (action: Action) => {
    setBusy(action);
    try {
      switch (action) {
        case "create":
          await request("", "POST", {});
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
          <Text className="text-sm text-red-600 dark:text-red-400">Could not load encryption keys.</Text>
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
            migrations={migrations}
            unfinished={!!unfinished}
            busy={busy}
            setConfirm={setConfirm}
            migrationError={migrationError}
            onRetry={() => void refresh()}
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
