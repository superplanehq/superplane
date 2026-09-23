import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { BookOpen } from "lucide-react";
import type { ReactNode } from "react";
import { formatDate } from "./formatDate";
import { AddVMRateForm } from "./priceBooksForms";
import { ModelsPanel, type PriceBookProvider } from "./priceBooksModelsPanel";
import { tableWrapClass, VMsTable } from "./priceBooksTables";
import type { PriceBookModelRate, PriceBooksResponse, PriceBookVMRate } from "./priceBooksApi";

export type PriceBooksTab = "models" | "machines";
export type { PriceBookProvider };

function isPriceBooksTab(value: string): value is PriceBooksTab {
  return value === "models" || value === "machines";
}

function PriceBooksHeader() {
  return (
    <div>
      <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">Price Books</h1>
      <Text className="mt-1 text-sm text-gray-500 dark:text-gray-400">
        Rates that SuperPlane uses to price hosted models and runner machines.
      </Text>
    </div>
  );
}

export function PriceBooksMessage({ message, action }: { message: string; action?: ReactNode }) {
  return (
    <div className="space-y-6">
      <PriceBooksHeader />
      <div className={`${tableWrapClass} p-8 text-center`}>
        <BookOpen size={24} className="mx-auto text-gray-400 dark:text-gray-500" />
        <Text className="mt-3 text-sm text-gray-600 dark:text-gray-400">{message}</Text>
        {action}
      </div>
    </div>
  );
}

type PriceBooksCatalogProps = {
  data: PriceBooksResponse;
  models: PriceBookModelRate[];
  vms: PriceBookVMRate[];
  tab: PriceBooksTab;
  provider: PriceBookProvider;
  saving: boolean;
  syncing: boolean;
  activating: boolean;
  deleting: boolean;
  versionLoading: boolean;
  pendingVersion?: string;
  onTabChange: (tab: PriceBooksTab) => void;
  onProviderChange: (provider: PriceBookProvider) => void;
  onVersionChange: (version: string) => void;
  onModelChange: (index: number, patch: Partial<PriceBookModelRate>) => void;
  onVMChange: (index: number, patch: Partial<PriceBookVMRate>) => boolean;
  onRemoveVM: (index: number) => void;
  onAddVM: (rate: PriceBookVMRate) => boolean;
  onSave: () => void;
  onSync: (provider: PriceBookProvider) => void;
  onActivate: () => void;
  onDelete: () => void;
};

export function PriceBooksCatalog(props: PriceBooksCatalogProps) {
  const isCurrent = props.data.version === props.data.current_version;
  const actionsDisabled = props.saving || props.syncing || props.activating || props.deleting || props.versionLoading;
  const canDelete = props.data.versions.length > 1 && !isCurrent;

  return (
    <div className="space-y-6">
      <PriceBooksToolbar
        data={props.data}
        isCurrent={isCurrent}
        activating={props.activating}
        deleting={props.deleting}
        mutating={props.saving || props.syncing || props.activating || props.deleting}
        actionsDisabled={actionsDisabled}
        canDelete={canDelete}
        pendingVersion={props.pendingVersion}
        onVersionChange={props.onVersionChange}
        onActivate={props.onActivate}
        onDelete={props.onDelete}
      />
      <Tabs
        value={props.tab}
        onValueChange={(nextTab) => {
          if (isPriceBooksTab(nextTab)) {
            props.onTabChange(nextTab);
          }
        }}
      >
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <TabsList>
            <TabsTrigger value="models">Models</TabsTrigger>
            <TabsTrigger value="machines">Machines</TabsTrigger>
          </TabsList>
          <Text className="text-xs text-gray-500 dark:text-gray-400">
            {props.tab === "models" ? "USD per 1 million tokens" : "USD per minute of machine time"}
          </Text>
        </div>
        <TabsContent value="models" className="mt-3 space-y-4">
          <ModelsPanel
            isCurrent={isCurrent}
            models={props.models}
            provider={props.provider}
            saving={props.saving}
            syncing={props.syncing}
            actionsDisabled={actionsDisabled}
            onProviderChange={props.onProviderChange}
            onModelChange={props.onModelChange}
            onSave={props.onSave}
            onSync={props.onSync}
          />
        </TabsContent>
        <TabsContent value="machines" className="mt-3 space-y-4">
          <VMsPanel
            isCurrent={isCurrent}
            vms={props.vms}
            saving={props.saving}
            actionsDisabled={actionsDisabled}
            onVMChange={props.onVMChange}
            onRemoveVM={props.onRemoveVM}
            onAddVM={props.onAddVM}
            onSave={props.onSave}
          />
        </TabsContent>
      </Tabs>
    </div>
  );
}

function PriceBooksToolbar({
  data,
  isCurrent,
  activating,
  deleting,
  mutating,
  actionsDisabled,
  canDelete,
  pendingVersion,
  onVersionChange,
  onActivate,
  onDelete,
}: {
  data: PriceBooksResponse;
  isCurrent: boolean;
  activating: boolean;
  deleting: boolean;
  mutating: boolean;
  actionsDisabled: boolean;
  canDelete: boolean;
  pendingVersion?: string;
  onVersionChange: (version: string) => void;
  onActivate: () => void;
  onDelete: () => void;
}) {
  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <PriceBooksHeader />
      <div className="flex flex-col gap-3 sm:items-end">
        <div className="flex flex-col gap-1">
          <Label htmlFor="admin-price-book-version">Version</Label>
          <Select value={pendingVersion ?? data.version} onValueChange={onVersionChange} disabled={mutating}>
            <SelectTrigger id="admin-price-book-version" data-testid="admin-price-book-version">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {data.versions.map((book) => (
                <SelectItem key={book.version} value={book.version}>
                  {book.version === data.current_version ? `${book.version} (current)` : book.version}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Text className="text-xs text-gray-500 dark:text-gray-400">Effective {formatDate(data.effective_at)}</Text>
        </div>
        <div className="flex flex-wrap gap-2 sm:justify-end">
          {!isCurrent && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              data-testid="admin-price-book-activate"
              disabled={actionsDisabled}
              onClick={onActivate}
            >
              {activating ? "Switching version..." : "Use this version"}
            </Button>
          )}
          {canDelete && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              data-testid="admin-price-book-delete"
              disabled={actionsDisabled}
              onClick={onDelete}
            >
              {deleting ? "Deleting version..." : "Delete this version"}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}

function VMsPanel({
  isCurrent,
  vms,
  saving,
  actionsDisabled,
  onVMChange,
  onRemoveVM,
  onAddVM,
  onSave,
}: {
  isCurrent: boolean;
  vms: PriceBookVMRate[];
  saving: boolean;
  actionsDisabled: boolean;
  onVMChange: (index: number, patch: Partial<PriceBookVMRate>) => boolean;
  onRemoveVM: (index: number) => void;
  onAddVM: (rate: PriceBookVMRate) => boolean;
  onSave: () => void;
}) {
  return (
    <>
      {isCurrent && (
        <div className="flex justify-end">
          <Button
            type="button"
            size="sm"
            data-testid="admin-price-book-save-vms"
            disabled={actionsDisabled}
            onClick={onSave}
          >
            {saving ? "Saving rates..." : "Save rates"}
          </Button>
        </div>
      )}
      <VMsTable rates={vms} editable={isCurrent && !actionsDisabled} onChange={onVMChange} onRemove={onRemoveVM} />
      {isCurrent && <AddVMRateForm disabled={actionsDisabled} onAdd={onAddVM} />}
    </>
  );
}
