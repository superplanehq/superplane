import { Link } from "@/components/Link/link";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { hostedProviderLabel } from "@/lib/hostedCredit";
import { ChevronDown, ChevronRight } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { EmptyRatesMessage, ModelsTable } from "./priceBooksTables";
import type { PriceBookModelRate } from "./priceBooksApi";

export type PriceBookProvider = "openrouter" | "anthropic" | "openai";

const UNUSED_PAGE_SIZE = 50;
const PROVIDERS: PriceBookProvider[] = ["openrouter", "anthropic", "openai"];

function isPriceBookProvider(value: string): value is PriceBookProvider {
  return value === "openrouter" || value === "anthropic" || value === "openai";
}

export function ModelsPanel({
  isCurrent,
  models,
  provider,
  saving,
  syncing,
  actionsDisabled,
  onProviderChange,
  onSave,
  onSync,
  onModelsChange,
}: {
  isCurrent: boolean;
  models: PriceBookModelRate[];
  provider: PriceBookProvider;
  saving: boolean;
  syncing: boolean;
  actionsDisabled: boolean;
  onProviderChange: (provider: PriceBookProvider) => void;
  onSave: () => Promise<boolean>;
  onSync: (provider: PriceBookProvider) => void;
  onModelsChange: (updater: (current: PriceBookModelRate[]) => PriceBookModelRate[]) => void;
}) {
  const providerRows = models.map((rate, index) => ({ rate, index })).filter(({ rate }) => rate.provider === provider);
  const selectedRows = providerRows.filter(({ rate }) => rate.selected);
  const unusedRows = providerRows.filter(({ rate }) => !rate.selected);
  const hasNoRates = providerRows.length === 0;

  const [editing, setEditing] = useState(false);
  const snapshotRef = useRef<PriceBookModelRate[]>([]);

  const leaveEditMode = () => {
    setEditing(false);
    snapshotRef.current = [];
  };

  const enterEditMode = () => {
    snapshotRef.current = models.map((rate) => ({ ...rate }));
    setEditing(true);
  };

  const handleRateChange = (index: number, updates: Partial<PriceBookModelRate>) => {
    onModelsChange((current) =>
      current.map((rate, rateIndex) => (rateIndex === index ? { ...rate, ...updates } : rate)),
    );
  };

  const handleCancelEdit = () => {
    onModelsChange(() => snapshotRef.current.map((rate) => ({ ...rate })));
    leaveEditMode();
  };

  useEffect(() => {
    if (!isCurrent) {
      leaveEditMode();
    }
  }, [isCurrent]);

  const isEditing = isCurrent && editing;

  return (
    <>
      <Tabs
        value={provider}
        onValueChange={(nextProvider) => {
          if (isPriceBookProvider(nextProvider) && nextProvider !== provider) {
            leaveEditMode();
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
          {provider === "openrouter" ? null : (
            <EditActionControl
              editing={isEditing}
              saving={saving}
              actionsDisabled={actionsDisabled}
              onCancel={handleCancelEdit}
              onEdit={enterEditMode}
              onSave={onSave}
              onSaved={leaveEditMode}
            />
          )}
        </div>
      )}
      {hasNoRates ? (
        <EmptyRatesMessage message="This version has no model rates for this provider." />
      ) : (
        <>
          <div>
            <Text className="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">Selected models</Text>
            {selectedRows.length > 0 ? (
              <ModelsTable rows={selectedRows} editing={isEditing} onRateChange={handleRateChange} />
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
          <UnusedModelsSection key={provider} rows={unusedRows} editing={isEditing} onRateChange={handleRateChange} />
        </>
      )}
    </>
  );
}

function EditActionControl({
  editing,
  saving,
  actionsDisabled,
  onCancel,
  onEdit,
  onSave,
  onSaved,
}: {
  editing: boolean;
  saving: boolean;
  actionsDisabled: boolean;
  onCancel: () => void;
  onEdit: () => void;
  onSave: () => Promise<boolean>;
  onSaved: () => void;
}) {
  if (!editing) {
    return (
      <Button type="button" variant="outline" size="sm" onClick={onEdit} disabled={actionsDisabled}>
        Edit rates
      </Button>
    );
  }

  return (
    <div className="flex gap-2">
      <Button type="button" variant="outline" size="sm" onClick={onCancel} disabled={saving}>
        Cancel
      </Button>
      <Button
        type="button"
        size="sm"
        data-testid="admin-price-book-save"
        disabled={actionsDisabled}
        onClick={() => {
          void onSave().then((saved) => {
            if (saved) {
              onSaved();
            }
          });
        }}
      >
        {saving ? "Saving rates..." : "Save rates"}
      </Button>
    </div>
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
            ? "The Anthropic API does not publish prices."
            : "The OpenAI API does not publish prices."}
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
  editing,
  onRateChange,
}: {
  rows: { rate: PriceBookModelRate; index: number }[];
  editing: boolean;
  onRateChange: (index: number, updates: Partial<PriceBookModelRate>) => void;
}) {
  const [open, setOpen] = useState(false);

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
            <ModelsTable rows={rows} pageSize={UNUSED_PAGE_SIZE} editing={editing} onRateChange={onRateChange} />
          </>
        ) : (
          <EmptyRatesMessage message="No unused model rates for this provider." />
        ))}
    </div>
  );
}
