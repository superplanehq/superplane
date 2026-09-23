import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { useState } from "react";
import { PriceBooksCatalog, PriceBooksMessage, type PriceBookProvider, type PriceBooksTab } from "./PriceBooksCatalog";
import { usePriceBookCatalog, usePriceBookEdits } from "./usePriceBooks";

export function PriceBooks() {
  const catalog = usePriceBookCatalog();
  const edits = usePriceBookEdits(catalog);
  const [tab, setTab] = useState<PriceBooksTab>("models");
  const [provider, setProvider] = useState<PriceBookProvider>("openrouter");
  const [activateOpen, setActivateOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);

  useReportPageReady(!catalog.loading);

  if (catalog.loading) {
    return (
      <div className="flex flex-col items-center space-y-4 py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-b border-gray-500 dark:border-gray-400"></div>
        <Text className="text-gray-500 dark:text-gray-400">Loading price books...</Text>
      </div>
    );
  }

  if (catalog.loadFailed || !catalog.data) {
    return (
      <PriceBooksMessage
        message="We could not load price books."
        action={
          <Button className="mt-4" variant="outline" size="sm" onClick={() => void catalog.loadPriceBooks()}>
            Try again
          </Button>
        }
      />
    );
  }

  const data = catalog.data;
  if (data.versions.length === 0) {
    return <PriceBooksMessage message="No price books in this installation." />;
  }

  return (
    <>
      <PriceBooksCatalog
        data={data}
        models={catalog.models}
        vms={catalog.vms}
        tab={tab}
        provider={provider}
        saving={edits.saving}
        syncing={edits.syncing}
        activating={edits.activating}
        deleting={edits.deleting}
        versionLoading={catalog.versionLoading}
        pendingVersion={catalog.pendingVersion}
        onTabChange={setTab}
        onProviderChange={setProvider}
        onVersionChange={(version) => {
          setActivateOpen(false);
          setDeleteOpen(false);
          void catalog.loadPriceBooks(version);
        }}
        onModelChange={edits.handleModelChange}
        onVMChange={edits.handleVMChange}
        onRemoveVM={edits.handleRemoveVM}
        onAddVM={edits.handleAddVM}
        onSave={edits.handleSave}
        onSync={edits.handleSync}
        onActivate={() => setActivateOpen(true)}
        onDelete={() => setDeleteOpen(true)}
      />
      <Dialog open={activateOpen} onClose={() => setActivateOpen(false)} size="md">
        <DialogTitle className="text-gray-800 dark:text-gray-100">Use this version</DialogTitle>
        <DialogDescription className="text-sm text-gray-600 dark:text-gray-400">
          <p>New hosted usage uses this version. Past usage keeps recorded costs.</p>
        </DialogDescription>
        <DialogActions>
          <Button
            data-testid="admin-price-book-activate-confirm"
            disabled={edits.activating || catalog.versionLoading}
            onClick={() => edits.handleActivate(data.version, () => setActivateOpen(false))}
          >
            {edits.activating ? "Switching version..." : "Use this version"}
          </Button>
          <Button variant="outline" onClick={() => setActivateOpen(false)}>
            Keep current version
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog open={deleteOpen} onClose={() => setDeleteOpen(false)} size="md">
        <DialogTitle className="text-gray-800 dark:text-gray-100">Delete this version</DialogTitle>
        <DialogDescription className="text-sm text-gray-600 dark:text-gray-400">
          <p>SuperPlane removes this version from the catalog. Past usage keeps recorded costs.</p>
        </DialogDescription>
        <DialogActions>
          <Button
            data-testid="admin-price-book-delete-confirm"
            disabled={edits.deleting || catalog.versionLoading}
            onClick={() => edits.handleDelete(data.version, () => setDeleteOpen(false))}
          >
            {edits.deleting ? "Deleting version..." : "Delete this version"}
          </Button>
          <Button variant="outline" onClick={() => setDeleteOpen(false)}>
            Keep this version
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
