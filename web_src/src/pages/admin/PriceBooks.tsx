import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { useState } from "react";
import { PriceBooksCatalog, PriceBooksMessage } from "./PriceBooksCatalog";
import { usePriceBookCatalog, usePriceBookEdits } from "./usePriceBooks";

type PriceBooksTab = "models" | "vms";

export function PriceBooks() {
  const catalog = usePriceBookCatalog();
  const edits = usePriceBookEdits(catalog);
  const [tab, setTab] = useState<PriceBooksTab>("models");
  const [activateOpen, setActivateOpen] = useState(false);

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
        saving={edits.saving}
        syncing={edits.syncing}
        activating={edits.activating}
        versionLoading={catalog.versionLoading}
        pendingVersion={catalog.pendingVersion}
        onTabChange={setTab}
        onVersionChange={(version) => {
          setActivateOpen(false);
          void catalog.loadPriceBooks(version);
        }}
        onModelChange={edits.handleModelChange}
        onVMChange={edits.handleVMChange}
        onAddModel={edits.handleAddModel}
        onAddVM={edits.handleAddVM}
        onSave={edits.handleSave}
        onSync={edits.handleSync}
        onActivate={() => setActivateOpen(true)}
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
    </>
  );
}
