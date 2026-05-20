import { Button } from "@/components/ui/button";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Input } from "@/components/Input/input";
import { Label } from "@/components/ui/label";
import { Text } from "@/components/Text/text";
import { Icon } from "@/components/Icon";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { Copy, Plus, Terminal, Trash2 } from "lucide-react";
import React, { useCallback, useEffect, useState } from "react";
import { formatDate } from "./formatDate";

type RunnerFleet = {
  id: string;
  name: string;
  created_at?: string;
};

type FleetCreatedPayload = RunnerFleet & {
  auth_token: string;
};

async function readErrorMessage(response: Response, fallback: string): Promise<string> {
  const text = await response.text();
  return text.trim() || fallback;
}

function CopyableValue({ label, value }: { label: string; value: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      showErrorToast("Failed to copy");
    }
  };

  return (
    <div className="space-y-1">
      <Label className="text-xs text-gray-500">{label}</Label>
      <div className="flex items-center gap-2">
        <code className="flex-1 text-xs bg-slate-100 border border-slate-200 rounded px-2 py-1.5 font-mono text-gray-800 break-all">
          {value}
        </code>
        <Button type="button" variant="outline" size="sm" onClick={handleCopy} className="shrink-0">
          <Icon name={copied ? "check" : "copy"} size="sm" />
        </Button>
      </div>
    </div>
  );
}

function FleetCreatedDialog({
  fleet,
  open,
  onClose,
}: {
  fleet: FleetCreatedPayload | null;
  open: boolean;
  onClose: () => void;
}) {
  if (!fleet) return null;

  return (
    <Dialog open={open} onClose={onClose} size="lg">
      <DialogTitle className="text-gray-800">Fleet registered</DialogTitle>
      <DialogDescription className="text-sm text-gray-600 mt-2 space-y-4">
        <p>
          <strong>{fleet.name}</strong> is ready. Copy these values now — the auth token is only shown once.
        </p>
        <CopyableValue label="Fleet ID (use as fleet_id on Runner nodes)" value={fleet.id} />
        <CopyableValue label="Auth token (SUPERPLANE_FLEET_AUTH_TOKEN on fleet-manager)" value={fleet.auth_token} />
        <div className="rounded-md bg-slate-50 border border-slate-200 px-3 py-2 text-xs text-gray-600 space-y-1">
          <p className="font-medium text-gray-700">Fleet-manager environment</p>
          <p>
            <code className="text-gray-800">SUPERPLANE_URL</code> — your SuperPlane API origin (e.g.{" "}
            <code className="text-gray-800">http://localhost:8000</code>)
          </p>
          <p>
            <code className="text-gray-800">SUPERPLANE_FLEET_AUTH_TOKEN</code> — the auth token above
          </p>
        </div>
      </DialogDescription>
      <DialogActions>
        <Button onClick={onClose}>Done</Button>
      </DialogActions>
    </Dialog>
  );
}

function DeleteFleetDialog({
  fleet,
  open,
  deleting,
  onClose,
  onConfirm,
}: {
  fleet: RunnerFleet | null;
  open: boolean;
  deleting: boolean;
  onClose: () => void;
  onConfirm: () => void;
}) {
  return (
    <Dialog open={open} onClose={onClose} size="md">
      <DialogTitle className="text-gray-800">Delete runner fleet</DialogTitle>
      <DialogDescription className="text-sm text-gray-600 mt-2">
        {fleet ? (
          <>
            Delete <strong>{fleet.name}</strong>? Queued runner tasks for this fleet will be removed. This cannot be
            undone.
          </>
        ) : null}
      </DialogDescription>
      <DialogActions>
        <Button variant="destructive" onClick={onConfirm} disabled={deleting}>
          {deleting ? "Deleting…" : "Delete fleet"}
        </Button>
        <Button variant="outline" onClick={onClose} disabled={deleting}>
          Cancel
        </Button>
      </DialogActions>
    </Dialog>
  );
}

const RunnerFleets: React.FC = () => {
  const [fleets, setFleets] = useState<RunnerFleet[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [name, setName] = useState("");
  const [createdFleet, setCreatedFleet] = useState<FleetCreatedPayload | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<RunnerFleet | null>(null);

  const loadFleets = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/admin/api/runner/fleets", { credentials: "include" });
      if (!res.ok) {
        showErrorToast(await readErrorMessage(res, "Failed to load runner fleets"));
        return;
      }
      const data = (await res.json()) as RunnerFleet[];
      setFleets(Array.isArray(data) ? data : []);
    } catch {
      showErrorToast("Failed to load runner fleets");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadFleets();
  }, [loadFleets]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    const trimmedName = name.trim();
    if (!trimmedName) {
      showErrorToast("Fleet name is required");
      return;
    }

    setCreating(true);
    try {
      const res = await fetch("/admin/api/runner/fleets", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: trimmedName }),
      });
      if (!res.ok) {
        showErrorToast(await readErrorMessage(res, "Failed to register fleet"));
        return;
      }
      const created = (await res.json()) as FleetCreatedPayload;
      setCreatedFleet(created);
      setName("");
      showSuccessToast(`Fleet "${created.name}" registered`);
      await loadFleets();
    } catch {
      showErrorToast("Failed to register fleet");
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      const res = await fetch(`/admin/api/runner/fleets/${deleteTarget.id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (!res.ok) {
        showErrorToast(await readErrorMessage(res, "Failed to delete fleet"));
        return;
      }
      showSuccessToast(`Fleet "${deleteTarget.name}" deleted`);
      setDeleteTarget(null);
      await loadFleets();
    } catch {
      showErrorToast("Failed to delete fleet");
    } finally {
      setDeleting(false);
    }
  };

  const copyFleetId = async (id: string) => {
    try {
      await navigator.clipboard.writeText(id);
      showSuccessToast("Fleet ID copied");
    } catch {
      showErrorToast("Failed to copy");
    }
  };

  if (loading && fleets.length === 0) {
    return (
      <div className="flex flex-col items-center space-y-4 py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b border-gray-500" />
        <Text className="text-gray-500">Loading runner fleets...</Text>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <FleetCreatedDialog fleet={createdFleet} open={createdFleet !== null} onClose={() => setCreatedFleet(null)} />
      <DeleteFleetDialog
        fleet={deleteTarget}
        open={deleteTarget !== null}
        deleting={deleting}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
      />

      <h1 className="text-xl font-semibold text-gray-900">Runner fleets</h1>

      <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 p-4">
        <h2 className="text-sm font-medium text-gray-800 mb-3 flex items-center gap-2">
          <Plus size={16} />
          Register fleet
        </h2>
        <form onSubmit={handleCreate} className="flex flex-col sm:flex-row sm:items-end gap-3">
          <div className="flex-1 space-y-1.5">
            <Label htmlFor="fleet-name">Name</Label>
            <Input
              id="fleet-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. local-dev-fleet"
              disabled={creating}
            />
          </div>
          <Button type="submit" disabled={creating} className="shrink-0 sm:mb-0.5">
            {creating ? "Registering…" : "Register fleet"}
          </Button>
        </form>
      </div>

      {fleets.length === 0 ? (
        <div className="text-center py-12 bg-white rounded-md shadow-sm outline outline-slate-950/10">
          <Text className="text-gray-500">No runner fleets yet. Register one above.</Text>
        </div>
      ) : (
        <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-100">
                <th className="text-left px-4 py-2.5 text-gray-500 font-medium">Name</th>
                <th className="text-left px-4 py-2.5 text-gray-500 font-medium">Fleet ID</th>
                <th className="text-left px-4 py-2.5 text-gray-500 font-medium">Created</th>
                <th className="text-right px-4 py-2.5 text-gray-500 font-medium w-24">Actions</th>
              </tr>
            </thead>
            <tbody>
              {fleets.map((fleet) => (
                <tr
                  key={fleet.id}
                  className="border-b border-slate-50 last:border-0 hover:bg-slate-50 transition-colors"
                >
                  <td className="px-4 py-2.5">
                    <span className="flex items-center gap-2 font-medium text-gray-800">
                      <Terminal size={14} className="text-gray-400 shrink-0" />
                      {fleet.name}
                    </span>
                  </td>
                  <td className="px-4 py-2.5">
                    <span className="flex items-center gap-1.5">
                      <code className="text-xs text-gray-600 font-mono">{fleet.id}</code>
                      <button
                        type="button"
                        onClick={() => void copyFleetId(fleet.id)}
                        className="p-1 rounded text-gray-400 hover:text-gray-700 hover:bg-slate-100"
                        title="Copy fleet ID"
                      >
                        <Copy size={14} />
                      </button>
                    </span>
                  </td>
                  <td className="px-4 py-2.5 text-gray-400 text-xs whitespace-nowrap">
                    {formatDate(fleet.created_at)}
                  </td>
                  <td className="px-4 py-2.5 text-right">
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="text-red-600 hover:text-red-700 hover:bg-red-50"
                      onClick={() => setDeleteTarget(fleet)}
                      title="Delete fleet"
                    >
                      <Trash2 size={16} />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};

export default RunnerFleets;
