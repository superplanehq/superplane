import { useCallback, useEffect, useState } from "react";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Table, TableHead, TableBody, TableRow, TableHeader, TableCell } from "@/components/Table/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { showSuccessToast } from "@/lib/toast";
import { billingCatalogRequest } from "./billingPlansApi";
import { CreditPackEditor } from "./CreditPackEditor";
import type { CreditPack, CreditPackInput } from "./creditPacksApi";

export function CreditPacks() {
  const [packs, setPacks] = useState<CreditPack[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState<CreditPack | "new" | null>(null);
  const [deleting, setDeleting] = useState<CreditPack | null>(null);
  const [removing, setRemoving] = useState(false);
  const [deleteError, setDeleteError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      setPacks(await billingCatalogRequest<CreditPack[]>("credit-packs"));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Could not load credit packs. Try again.");
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);

  const save = async (input: CreditPackInput) => {
    if (!editing) return;
    await billingCatalogRequest(
      editing === "new" ? "credit-packs" : `credit-packs/${editing.id}`,
      editing === "new" ? "POST" : "PUT",
      input,
    );
    setEditing(null);
    showSuccessToast("Credit pack saved");
    await load();
  };
  const remove = async () => {
    if (!deleting || removing) return;
    setRemoving(true);
    setDeleteError("");
    try {
      await billingCatalogRequest(`credit-packs/${deleting.id}`, "DELETE");
      setDeleting(null);
      showSuccessToast("Credit pack deleted");
      await load();
    } catch (cause) {
      setDeleteError(cause instanceof Error ? cause.message : "Could not delete the credit pack. Try again.");
    } finally {
      setRemoving(false);
    }
  };

  return (
    <section className="mt-6 space-y-4" aria-labelledby="credit-packs-heading">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-1">
          <h2 id="credit-packs-heading" className="text-base font-semibold">
            Credit packs
          </h2>
          <p className="text-sm text-muted-foreground">
            Set the credit amounts available for purchase across all organizations.
          </p>
        </div>
        <Button disabled={loading || !!error} onClick={() => setEditing("new")}>
          <Plus aria-hidden="true" />
          Create pack
        </Button>
      </div>
      {error ? (
        <div role="alert" className="flex items-center justify-between gap-4 rounded-lg border p-4 text-sm">
          {error}
          <Button variant="outline" onClick={() => void load()}>
            Try again
          </Button>
        </div>
      ) : loading ? (
        <p className="py-8 text-center text-sm text-muted-foreground">Loading credit packs...</p>
      ) : (
        <CreditPacksTable
          packs={packs}
          onEdit={setEditing}
          onDelete={(pack) => {
            setDeleteError("");
            setDeleting(pack);
          }}
        />
      )}
      {editing && (
        <CreditPackEditor pack={editing === "new" ? null : editing} onSave={save} onCancel={() => setEditing(null)} />
      )}
      {deleting && (
        <DeleteCreditPackDialog
          pack={deleting}
          removing={removing}
          error={deleteError}
          onCancel={() => setDeleting(null)}
          onDelete={() => void remove()}
        />
      )}
    </section>
  );
}

function CreditPacksTable({
  packs,
  onEdit,
  onDelete,
}: {
  packs: CreditPack[];
  onEdit: (pack: CreditPack) => void;
  onDelete: (pack: CreditPack) => void;
}) {
  return (
    <div className="overflow-x-auto rounded-lg border px-6">
      <Table aria-label="Credit packs" bleed>
        <TableHead>
          <TableRow>
            <TableHeader>Pack</TableHeader>
            <TableHeader>Credit amount</TableHeader>
            <TableHeader>Polar product ID</TableHeader>
            <TableHeader>Status</TableHeader>
            <TableHeader>
              <span className="sr-only">Actions</span>
            </TableHeader>
          </TableRow>
        </TableHead>
        <TableBody>
          {packs.length === 0 && (
            <TableRow>
              <TableCell colSpan={5} className="text-center text-muted-foreground">
                No credit packs configured. Create a pack to set purchase amounts.
              </TableCell>
            </TableRow>
          )}
          {packs.map((pack) => (
            <TableRow key={pack.id}>
              <TableCell className="font-medium">{pack.name}</TableCell>
              <TableCell>
                {pack.amount_cents === 0
                  ? "Custom amount"
                  : (pack.amount_cents / 100).toLocaleString("en-US", { style: "currency", currency: "USD" })}
              </TableCell>
              <TableCell>
                {pack.polar_product_id ? (
                  <code className="text-xs">{pack.polar_product_id}</code>
                ) : (
                  <span className="text-muted-foreground">Not linked</span>
                )}
              </TableCell>
              <TableCell>
                <Badge variant="outline">
                  {!pack.enabled ? "Disabled" : pack.polar_product_id ? "Enabled" : "Not linked"}
                </Badge>
              </TableCell>
              <TableCell>
                <div className="flex justify-end gap-1">
                  <Button size="icon-sm" variant="ghost" aria-label={`Edit ${pack.name}`} onClick={() => onEdit(pack)}>
                    <Pencil aria-hidden="true" />
                  </Button>
                  <Button
                    size="icon-sm"
                    variant="ghost"
                    aria-label={`Delete ${pack.name}`}
                    onClick={() => {
                      onDelete(pack);
                    }}
                  >
                    <Trash2 aria-hidden="true" />
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function DeleteCreditPackDialog({
  pack,
  removing,
  error,
  onCancel,
  onDelete,
}: {
  pack: CreditPack;
  removing: boolean;
  error: string;
  onCancel: () => void;
  onDelete: () => void;
}) {
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !removing) onCancel();
      }}
    >
      <DialogContent showCloseButton={!removing}>
        <DialogHeader>
          <DialogTitle>Delete {pack.name}?</DialogTitle>
          <DialogDescription>
            This pack will no longer appear in organization billing. Completed purchases keep their credits.
          </DialogDescription>
        </DialogHeader>
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
        <DialogFooter>
          <Button variant="outline" disabled={removing} onClick={onCancel}>
            Keep pack
          </Button>
          <Button variant="destructive" disabled={removing} onClick={onDelete}>
            {removing ? "Deleting..." : "Delete pack"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
