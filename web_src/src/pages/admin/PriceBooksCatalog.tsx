import { Link } from "@/components/Link/link";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { BookOpen } from "lucide-react";
import type { ReactNode } from "react";
import { formatDate } from "./formatDate";
import { AddModelRateForm, AddVMRateForm } from "./priceBooksForms";
import { EmptyRatesMessage, ModelsTable, tableWrapClass, VMsTable } from "./priceBooksTables";
import type { PriceBookModelRate, PriceBooksResponse, PriceBookVMRate } from "./priceBooksApi";

type PriceBooksTab = "models" | "vms";

function isPriceBooksTab(value: string): value is PriceBooksTab {
  return value === "models" || value === "vms";
}

function PriceBooksHeader() {
  return (
    <div>
      <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">Price Books</h1>
      <Text className="mt-1 text-sm text-gray-500 dark:text-gray-400">
        Rates that SuperPlane uses to price hosted models and runner VMs.
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
  saving: boolean;
  syncing: boolean;
  activating: boolean;
  versionLoading: boolean;
  pendingVersion?: string;
  onTabChange: (tab: PriceBooksTab) => void;
  onVersionChange: (version: string) => void;
  onModelChange: (index: number, patch: Partial<PriceBookModelRate>) => void;
  onVMChange: (index: number, patch: Partial<PriceBookVMRate>) => boolean;
  onRemoveVM: (index: number) => void;
  onAddModel: (rate: PriceBookModelRate) => boolean;
  onAddVM: (rate: PriceBookVMRate) => boolean;
  onSave: () => void;
  onSync: () => void;
  onActivate: () => void;
};

export function PriceBooksCatalog(props: PriceBooksCatalogProps) {
  const isCurrent = props.data.version === props.data.current_version;
  const actionsDisabled = props.saving || props.syncing || props.activating || props.versionLoading;

  return (
    <div className="space-y-6">
      <PriceBooksToolbar
        data={props.data}
        isCurrent={isCurrent}
        activating={props.activating}
        mutating={props.saving || props.syncing || props.activating}
        actionsDisabled={actionsDisabled}
        pendingVersion={props.pendingVersion}
        onVersionChange={props.onVersionChange}
        onActivate={props.onActivate}
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
            <TabsTrigger value="vms">VMs</TabsTrigger>
          </TabsList>
          <Text className="text-xs text-gray-500 dark:text-gray-400">
            {props.tab === "models" ? "USD per 1 million tokens" : "USD per minute of machine time"}
          </Text>
        </div>
        <TabsContent value="models" className="mt-3 space-y-4">
          <ModelsPanel
            isCurrent={isCurrent}
            models={props.models}
            saving={props.saving}
            syncing={props.syncing}
            actionsDisabled={actionsDisabled}
            onModelChange={props.onModelChange}
            onAddModel={props.onAddModel}
            onSave={props.onSave}
            onSync={props.onSync}
          />
        </TabsContent>
        <TabsContent value="vms" className="mt-3 space-y-4">
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
  mutating,
  actionsDisabled,
  pendingVersion,
  onVersionChange,
  onActivate,
}: {
  data: PriceBooksResponse;
  isCurrent: boolean;
  activating: boolean;
  mutating: boolean;
  actionsDisabled: boolean;
  pendingVersion?: string;
  onVersionChange: (version: string) => void;
  onActivate: () => void;
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
      </div>
    </div>
  );
}

function ModelsPanel({
  isCurrent,
  models,
  saving,
  syncing,
  actionsDisabled,
  onModelChange,
  onAddModel,
  onSave,
  onSync,
}: {
  isCurrent: boolean;
  models: PriceBookModelRate[];
  saving: boolean;
  syncing: boolean;
  actionsDisabled: boolean;
  onModelChange: (index: number, patch: Partial<PriceBookModelRate>) => void;
  onAddModel: (rate: PriceBookModelRate) => boolean;
  onSave: () => void;
  onSync: () => void;
}) {
  const hasNoRates = models.length === 0;

  const selectedRows = models.map((rate, index) => ({ rate, index })).filter(({ rate }) => rate.selected);
  const otherRows = models.map((rate, index) => ({ rate, index })).filter(({ rate }) => !rate.selected);

  return (
    <>
      {isCurrent && (
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              data-testid="admin-price-book-sync"
              disabled={actionsDisabled}
              onClick={onSync}
            >
              {syncing ? "Updating model rates..." : "Update model rates"}
            </Button>
            <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              Creates a new current version from enabled provider catalogs. Past usage keeps recorded costs.
            </Text>
          </div>
          <Button
            type="button"
            size="sm"
            data-testid="admin-price-book-save"
            disabled={actionsDisabled}
            onClick={onSave}
          >
            {saving ? "Saving rates..." : "Save rates"}
          </Button>
        </div>
      )}
      {hasNoRates ? (
        <EmptyRatesMessage message="This version has no model rates." />
      ) : (
        <>
          <div>
            <Text className="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">Selected models</Text>
            {selectedRows.length > 0 ? (
              <ModelsTable rows={selectedRows} editable={isCurrent && !actionsDisabled} onChange={onModelChange} />
            ) : (
              <EmptyRatesMessage
                message="No models are selected in Hosted LLM settings."
                action={
                  <Link
                    href="/admin/settings"
                    className="mt-3 inline-block text-sm font-medium text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
                  >
                    Open Hosted LLM settings
                  </Link>
                }
              />
            )}
          </div>
          <div>
            <Text className="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">Other models</Text>
            {otherRows.length > 0 ? (
              <ModelsTable rows={otherRows} editable={isCurrent && !actionsDisabled} onChange={onModelChange} />
            ) : (
              <EmptyRatesMessage message="No other model rates in this version." />
            )}
          </div>
        </>
      )}
      {isCurrent && <AddModelRateForm disabled={actionsDisabled} onAdd={onAddModel} />}
    </>
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
