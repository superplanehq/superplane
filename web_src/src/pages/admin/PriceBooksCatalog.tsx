import { Link } from "@/components/Link/link";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { hostedProviderLabel } from "@/lib/hostedCredit";
import { BookOpen, ChevronDown, ChevronRight } from "lucide-react";
import { useState, type ReactNode } from "react";
import AdminPagination from "./AdminPagination";
import { formatDate } from "./formatDate";
import { AddModelRateForm, AddVMRateForm } from "./priceBooksForms";
import { EmptyRatesMessage, ModelsTable, tableWrapClass, VMsTable } from "./priceBooksTables";
import type { PriceBookModelRate, PriceBooksResponse, PriceBookVMRate } from "./priceBooksApi";

export type PriceBooksTab = "models" | "machines";
export type PriceBookProvider = "openrouter" | "anthropic" | "openai";

const UNUSED_PAGE_SIZE = 50;
const PROVIDERS: PriceBookProvider[] = ["openrouter", "anthropic", "openai"];

function isPriceBooksTab(value: string): value is PriceBooksTab {
  return value === "models" || value === "machines";
}

function isPriceBookProvider(value: string): value is PriceBookProvider {
  return value === "openrouter" || value === "anthropic" || value === "openai";
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
  onAddModel: (rate: PriceBookModelRate) => boolean;
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
            onAddModel={props.onAddModel}
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

function ModelsPanel({
  isCurrent,
  models,
  provider,
  saving,
  syncing,
  actionsDisabled,
  onProviderChange,
  onModelChange,
  onAddModel,
  onSave,
  onSync,
}: {
  isCurrent: boolean;
  models: PriceBookModelRate[];
  provider: PriceBookProvider;
  saving: boolean;
  syncing: boolean;
  actionsDisabled: boolean;
  onProviderChange: (provider: PriceBookProvider) => void;
  onModelChange: (index: number, patch: Partial<PriceBookModelRate>) => void;
  onAddModel: (rate: PriceBookModelRate) => boolean;
  onSave: () => void;
  onSync: (provider: PriceBookProvider) => void;
}) {
  const providerRows = models.map((rate, index) => ({ rate, index })).filter(({ rate }) => rate.provider === provider);
  const selectedRows = providerRows.filter(({ rate }) => rate.selected);
  const unusedRows = providerRows.filter(({ rate }) => !rate.selected);
  const hasNoRates = providerRows.length === 0;

  return (
    <>
      <Tabs
        value={provider}
        onValueChange={(nextProvider) => {
          if (isPriceBookProvider(nextProvider)) {
            onProviderChange(nextProvider);
          }
        }}
      >
        <TabsList>
          {PROVIDERS.map((item) => (
            <TabsTrigger key={item} value={item}>
              {hostedProviderLabel(item)}
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>
      {isCurrent && (
        <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
          <ProviderUpdateControl
            provider={provider}
            syncing={syncing}
            actionsDisabled={actionsDisabled}
            onSync={onSync}
          />
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
        <EmptyRatesMessage message="This version has no model rates for this provider." />
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
          <UnusedModelsSection
            key={provider}
            rows={unusedRows}
            editable={isCurrent && !actionsDisabled}
            onChange={onModelChange}
          />
        </>
      )}
      {isCurrent && <AddModelRateForm provider={provider} disabled={actionsDisabled} onAdd={onAddModel} />}
    </>
  );
}

function ProviderUpdateControl({
  provider,
  syncing,
  actionsDisabled,
  onSync,
}: {
  provider: PriceBookProvider;
  syncing: boolean;
  actionsDisabled: boolean;
  onSync: (provider: PriceBookProvider) => void;
}) {
  if (provider !== "openrouter") {
    return (
      <div>
        <Button type="button" variant="outline" size="sm" disabled data-testid="admin-price-book-sync-disabled">
          Update model rates
        </Button>
        <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {provider === "anthropic"
            ? "The Anthropic API does not publish prices. Edit rates here."
            : "The OpenAI API does not publish prices. Edit rates here."}
        </Text>
      </div>
    );
  }

  return (
    <div>
      <Button
        type="button"
        variant="outline"
        size="sm"
        data-testid="admin-price-book-sync"
        disabled={actionsDisabled}
        onClick={() => onSync(provider)}
      >
        {syncing ? "Updating model rates..." : "Update model rates"}
      </Button>
      <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
        Creates a new current version from the OpenRouter catalog. Past usage keeps recorded costs.
      </Text>
    </div>
  );
}

function UnusedModelsSection({
  rows,
  editable,
  onChange,
}: {
  rows: { rate: PriceBookModelRate; index: number }[];
  editable: boolean;
  onChange: (index: number, patch: Partial<PriceBookModelRate>) => void;
}) {
  const [open, setOpen] = useState(false);
  const [offset, setOffset] = useState(0);
  const pageRows = rows.slice(offset, offset + UNUSED_PAGE_SIZE);

  return (
    <div>
      <button
        type="button"
        className="mb-2 flex items-center gap-1 text-sm font-medium text-gray-700 dark:text-gray-300"
        data-testid="admin-price-book-unused-toggle"
        onClick={() => setOpen((current) => !current)}
      >
        {open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
        Unused models ({rows.length})
      </button>
      {open &&
        (rows.length > 0 ? (
          <>
            <ModelsTable rows={pageRows} editable={editable} onChange={onChange} />
            <AdminPagination offset={offset} total={rows.length} pageSize={UNUSED_PAGE_SIZE} onPageChange={setOffset} />
          </>
        ) : (
          <EmptyRatesMessage message="No unused model rates for this provider." />
        ))}
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
