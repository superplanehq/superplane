import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { BookOpen } from "lucide-react";
import type { ReactNode } from "react";
import { formatDate } from "./formatDate";
import { AddModelRateForm, AddVMRateForm } from "./priceBooksForms";
import { ModelsTable, tableWrapClass, VMsTable } from "./priceBooksTables";
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
  onTabChange: (tab: PriceBooksTab) => void;
  onVersionChange: (version: string) => void;
  onModelChange: (index: number, patch: Partial<PriceBookModelRate>) => void;
  onVMChange: (index: number, micros: number) => void;
  onAddModel: (rate: PriceBookModelRate) => boolean;
  onAddVM: (rate: PriceBookVMRate) => boolean;
  onSave: () => void;
  onSync: () => void;
  onActivate: () => void;
};

export function PriceBooksCatalog(props: PriceBooksCatalogProps) {
  const isCurrent = props.data.version === props.data.current_version;

  return (
    <div className="space-y-6">
      <PriceBooksToolbar
        data={props.data}
        isCurrent={isCurrent}
        activating={props.activating}
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
            onVMChange={props.onVMChange}
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
  onVersionChange,
  onActivate,
}: {
  data: PriceBooksResponse;
  isCurrent: boolean;
  activating: boolean;
  onVersionChange: (version: string) => void;
  onActivate: () => void;
}) {
  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <PriceBooksHeader />
      <div className="flex flex-col gap-3 sm:items-end">
        <div className="flex flex-col gap-1">
          <Label htmlFor="admin-price-book-version">Version</Label>
          <Select value={data.version} onValueChange={onVersionChange}>
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
            disabled={activating}
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
  onModelChange,
  onAddModel,
  onSave,
  onSync,
}: {
  isCurrent: boolean;
  models: PriceBookModelRate[];
  saving: boolean;
  syncing: boolean;
  onModelChange: (index: number, patch: Partial<PriceBookModelRate>) => void;
  onAddModel: (rate: PriceBookModelRate) => boolean;
  onSave: () => void;
  onSync: () => void;
}) {
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
              disabled={syncing || saving}
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
            disabled={saving || syncing}
            onClick={onSave}
          >
            {saving ? "Saving rates..." : "Save rates"}
          </Button>
        </div>
      )}
      <ModelsTable rates={models} editable={isCurrent} onChange={onModelChange} />
      {isCurrent && <AddModelRateForm onAdd={onAddModel} />}
    </>
  );
}

function VMsPanel({
  isCurrent,
  vms,
  saving,
  onVMChange,
  onAddVM,
  onSave,
}: {
  isCurrent: boolean;
  vms: PriceBookVMRate[];
  saving: boolean;
  onVMChange: (index: number, micros: number) => void;
  onAddVM: (rate: PriceBookVMRate) => boolean;
  onSave: () => void;
}) {
  return (
    <>
      {isCurrent && (
        <div className="flex justify-end">
          <Button type="button" size="sm" data-testid="admin-price-book-save-vms" disabled={saving} onClick={onSave}>
            {saving ? "Saving rates..." : "Save rates"}
          </Button>
        </div>
      )}
      <VMsTable rates={vms} editable={isCurrent} onChange={onVMChange} />
      {isCurrent && <AddVMRateForm onAdd={onAddVM} />}
    </>
  );
}
