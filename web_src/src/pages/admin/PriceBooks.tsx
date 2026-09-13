import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { BookOpen } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { formatDate } from "./formatDate";
import {
  centsToUsdInput,
  formatMatchMode,
  formatMicrosPerSecondUsdPerMinute,
  usdInputToCents,
} from "./priceBookFormat";
import {
  activatePriceBook,
  fetchPriceBooks,
  savePriceBooks,
  syncPriceBooks,
  type PriceBookModelRate,
  type PriceBooksResponse,
  type PriceBookVMRate,
} from "./priceBooksApi";

type PriceBooksTab = "models" | "vms";

function isPriceBooksTab(value: string): value is PriceBooksTab {
  return value === "models" || value === "vms";
}

const tableWrapClass =
  "bg-white rounded-md shadow-sm outline outline-slate-950/10 overflow-hidden dark:bg-gray-900 dark:outline-gray-700/70";
const headerCellClass = "text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400";
const numericHeaderCellClass = `${headerCellClass} text-right`;
const bodyCellClass = "px-4 py-2.5 text-gray-700 dark:text-gray-300";
const numericCellClass = `${bodyCellClass} text-right tabular-nums`;
const rowClass = "border-b border-slate-50 last:border-0 dark:border-gray-800/70";

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

function PriceBooksMessage({ message, action }: { message: string; action?: ReactNode }) {
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

function EmptyRatesMessage({ message }: { message: string }) {
  return (
    <div className={`${tableWrapClass} p-8 text-center`}>
      <BookOpen size={24} className="mx-auto text-gray-400 dark:text-gray-500" />
      <Text className="mt-3 text-sm text-gray-600 dark:text-gray-400">{message}</Text>
    </div>
  );
}

function UsdRateInput({
  id,
  cents,
  disabled,
  onCommit,
}: {
  id: string;
  cents: number;
  disabled: boolean;
  onCommit: (cents: number) => void;
}) {
  const [text, setText] = useState(centsToUsdInput(cents));
  useEffect(() => {
    setText(centsToUsdInput(cents));
  }, [cents]);

  return (
    <Input
      id={id}
      type="number"
      min="0"
      step="0.01"
      disabled={disabled}
      className="h-8 text-right tabular-nums"
      value={text}
      onChange={(event) => setText(event.target.value)}
      onBlur={() => onCommit(usdInputToCents(text))}
    />
  );
}

function ModelsTable({
  rates,
  editable,
  onChange,
}: {
  rates: PriceBookModelRate[];
  editable: boolean;
  onChange: (index: number, patch: Partial<PriceBookModelRate>) => void;
}) {
  if (rates.length === 0) {
    return <EmptyRatesMessage message="This version has no model rates." />;
  }

  return (
    <div className={`${tableWrapClass} overflow-x-auto`}>
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-100 dark:border-gray-700/70">
            <th className={headerCellClass}>Match</th>
            <th className={headerCellClass}>Mode</th>
            <th className={numericHeaderCellClass}>Input</th>
            <th className={numericHeaderCellClass}>Output</th>
            <th className={numericHeaderCellClass}>Cache read</th>
            <th className={numericHeaderCellClass}>Cache write</th>
            <th className={numericHeaderCellClass}>Reasoning</th>
          </tr>
        </thead>
        <tbody>
          {rates.map((rate, index) => (
            <tr key={`${rate.match_key}:${rate.match_mode}`} className={rowClass}>
              <td className={`${bodyCellClass} font-mono text-xs`}>{rate.match_key}</td>
              <td className={bodyCellClass}>{formatMatchMode(rate.match_mode)}</td>
              {(
                [
                  ["input_cents_per_million", rate.input_cents_per_million],
                  ["output_cents_per_million", rate.output_cents_per_million],
                  ["cache_read_cents_per_million", rate.cache_read_cents_per_million],
                  ["cache_write_cents_per_million", rate.cache_write_cents_per_million],
                  ["reasoning_cents_per_million", rate.reasoning_cents_per_million],
                ] as const
              ).map(([field, value]) => (
                <td key={field} className={numericCellClass}>
                  {editable ? (
                    <UsdRateInput
                      id={`price-book-model-${rate.match_key}-${rate.match_mode}-${field}`}
                      cents={value}
                      disabled={!editable}
                      onCommit={(cents) => onChange(index, { [field]: cents })}
                    />
                  ) : (
                    `$${centsToUsdInput(value)}`
                  )}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function VMsTable({
  rates,
  editable,
  onChange,
}: {
  rates: PriceBookVMRate[];
  editable: boolean;
  onChange: (index: number, micros: number) => void;
}) {
  if (rates.length === 0) {
    return <EmptyRatesMessage message="This version has no VM rates." />;
  }

  return (
    <div className={tableWrapClass}>
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-100 dark:border-gray-700/70">
            <th className={headerCellClass}>Machine type</th>
            <th className={numericHeaderCellClass}>Micros per second</th>
            <th className={numericHeaderCellClass}>Rate</th>
          </tr>
        </thead>
        <tbody>
          {rates.map((rate, index) => (
            <tr key={`${rate.match_key}:${rate.match_mode}`} className={rowClass}>
              <td className={`${bodyCellClass} font-mono text-xs`}>{rate.match_key}</td>
              <td className={numericCellClass}>
                {editable ? (
                  <Input
                    id={`price-book-vm-${rate.match_key}-micros`}
                    type="number"
                    min="0"
                    step="1"
                    className="h-8 text-right font-mono text-xs tabular-nums"
                    value={rate.micros_per_second}
                    onChange={(event) => {
                      const parsed = Number.parseInt(event.target.value, 10);
                      onChange(index, Number.isFinite(parsed) && parsed >= 0 ? parsed : 0);
                    }}
                  />
                ) : (
                  <span className="font-mono text-xs">{rate.micros_per_second}</span>
                )}
              </td>
              <td className={numericCellClass}>{formatMicrosPerSecondUsdPerMinute(rate.micros_per_second)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function AddModelRateForm({ onAdd }: { onAdd: (rate: PriceBookModelRate) => boolean }) {
  const [matchKey, setMatchKey] = useState("");
  const [matchMode, setMatchMode] = useState("prefix");
  const [input, setInput] = useState("0.00");
  const [output, setOutput] = useState("0.00");

  return (
    <form
      className="flex flex-col gap-3 lg:flex-row lg:flex-wrap lg:items-end"
      onSubmit={(event) => {
        event.preventDefault();
        const added = onAdd({
          match_key: matchKey.trim(),
          match_mode: matchMode,
          input_cents_per_million: usdInputToCents(input),
          output_cents_per_million: usdInputToCents(output),
          cache_read_cents_per_million: 0,
          cache_write_cents_per_million: 0,
          reasoning_cents_per_million: 0,
        });
        if (added) {
          setMatchKey("");
          setInput("0.00");
          setOutput("0.00");
        }
      }}
    >
      <div className="min-w-40 flex-1">
        <Label htmlFor="price-book-add-model-key">Match key</Label>
        <Input
          id="price-book-add-model-key"
          className="mt-1 font-mono text-xs"
          value={matchKey}
          onChange={(event) => setMatchKey(event.target.value)}
        />
      </div>
      <div className="w-40">
        <Label htmlFor="price-book-add-model-mode">Mode</Label>
        <Select value={matchMode} onValueChange={setMatchMode}>
          <SelectTrigger id="price-book-add-model-mode" className="mt-1">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="prefix">Prefix</SelectItem>
            <SelectItem value="family">Family</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div className="w-28">
        <Label htmlFor="price-book-add-model-input">Input</Label>
        <Input
          id="price-book-add-model-input"
          type="number"
          min="0"
          step="0.01"
          className="mt-1 text-right"
          value={input}
          onChange={(event) => setInput(event.target.value)}
        />
      </div>
      <div className="w-28">
        <Label htmlFor="price-book-add-model-output">Output</Label>
        <Input
          id="price-book-add-model-output"
          type="number"
          min="0"
          step="0.01"
          className="mt-1 text-right"
          value={output}
          onChange={(event) => setOutput(event.target.value)}
        />
      </div>
      <Button type="submit" variant="outline" size="sm">
        Add model rate
      </Button>
    </form>
  );
}

function AddVMRateForm({ onAdd }: { onAdd: (rate: PriceBookVMRate) => boolean }) {
  const [matchKey, setMatchKey] = useState("");
  const [micros, setMicros] = useState("0");

  return (
    <form
      className="flex flex-col gap-3 sm:flex-row sm:items-end"
      onSubmit={(event) => {
        event.preventDefault();
        const parsed = Number.parseInt(micros, 10);
        const added = onAdd({
          match_key: matchKey.trim(),
          match_mode: "exact",
          micros_per_second: Number.isFinite(parsed) && parsed >= 0 ? parsed : 0,
        });
        if (added) {
          setMatchKey("");
          setMicros("0");
        }
      }}
    >
      <div className="min-w-40 flex-1">
        <Label htmlFor="price-book-add-vm-key">Machine type</Label>
        <Input
          id="price-book-add-vm-key"
          className="mt-1 font-mono text-xs"
          value={matchKey}
          onChange={(event) => setMatchKey(event.target.value)}
        />
      </div>
      <div className="w-40">
        <Label htmlFor="price-book-add-vm-micros">Micros per second</Label>
        <Input
          id="price-book-add-vm-micros"
          type="number"
          min="0"
          step="1"
          className="mt-1 text-right font-mono text-xs"
          value={micros}
          onChange={(event) => setMicros(event.target.value)}
        />
      </div>
      <Button type="submit" variant="outline" size="sm">
        Add VM rate
      </Button>
    </form>
  );
}

function PriceBooksCatalog({
  data,
  models,
  vms,
  tab,
  saving,
  syncing,
  activating,
  onTabChange,
  onVersionChange,
  onModelChange,
  onVMChange,
  onAddModel,
  onAddVM,
  onSave,
  onSync,
  onActivate,
}: {
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
}) {
  const isCurrent = data.version === data.current_version;

  return (
    <div className="space-y-6">
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

      <Tabs
        value={tab}
        onValueChange={(nextTab) => {
          if (isPriceBooksTab(nextTab)) {
            onTabChange(nextTab);
          }
        }}
      >
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <TabsList>
            <TabsTrigger value="models">Models</TabsTrigger>
            <TabsTrigger value="vms">VMs</TabsTrigger>
          </TabsList>
          <Text className="text-xs text-gray-500 dark:text-gray-400">
            {tab === "models" ? "USD per 1 million tokens" : "USD per minute of machine time"}
          </Text>
        </div>
        <TabsContent value="models" className="mt-3 space-y-4">
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
        </TabsContent>
        <TabsContent value="vms" className="mt-3 space-y-4">
          {isCurrent && (
            <div className="flex justify-end">
              <Button
                type="button"
                size="sm"
                data-testid="admin-price-book-save-vms"
                disabled={saving}
                onClick={onSave}
              >
                {saving ? "Saving rates..." : "Save rates"}
              </Button>
            </div>
          )}
          <VMsTable rates={vms} editable={isCurrent} onChange={onVMChange} />
          {isCurrent && <AddVMRateForm onAdd={onAddVM} />}
        </TabsContent>
      </Tabs>
    </div>
  );
}

export function PriceBooks() {
  const [data, setData] = useState<PriceBooksResponse | null>(null);
  const [models, setModels] = useState<PriceBookModelRate[]>([]);
  const [vms, setVMs] = useState<PriceBookVMRate[]>([]);
  const [tab, setTab] = useState<PriceBooksTab>("models");
  const [loading, setLoading] = useState(true);
  const [loadFailed, setLoadFailed] = useState(false);
  const [saving, setSaving] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [activating, setActivating] = useState(false);
  const [activateOpen, setActivateOpen] = useState(false);
  const loadAbort = useRef<AbortController | null>(null);
  const loadGeneration = useRef(0);

  const applyCatalog = useCallback((payload: PriceBooksResponse) => {
    setData(payload);
    setModels(payload.models.map((rate) => ({ ...rate })));
    setVMs(payload.vms.map((rate) => ({ ...rate })));
  }, []);

  const loadPriceBooks = useCallback(
    async (version?: string) => {
      const isFirstLoad = version === undefined;
      if (isFirstLoad) {
        setLoading(true);
        setLoadFailed(false);
      }

      loadAbort.current?.abort();
      const controller = new AbortController();
      loadAbort.current = controller;
      const generation = ++loadGeneration.current;

      try {
        const payload = await fetchPriceBooks(version, controller.signal);
        if (generation !== loadGeneration.current) {
          return;
        }

        applyCatalog(payload);
        setLoadFailed(false);
      } catch (error) {
        if (generation !== loadGeneration.current || controller.signal.aborted) {
          return;
        }

        showErrorToast(error instanceof Error ? error.message : "Failed to load price books");
        if (isFirstLoad) {
          setLoadFailed(true);
          setData(null);
        }
      } finally {
        if (isFirstLoad && generation === loadGeneration.current) {
          setLoading(false);
        }
      }
    },
    [applyCatalog],
  );

  useEffect(() => {
    void loadPriceBooks();
    return () => {
      loadAbort.current?.abort();
    };
  }, [loadPriceBooks]);

  useReportPageReady(!loading);

  const modelKeys = useMemo(() => new Set(models.map((rate) => `${rate.match_key}:${rate.match_mode}`)), [models]);
  const vmKeys = useMemo(() => new Set(vms.map((rate) => `${rate.match_key}:${rate.match_mode}`)), [vms]);

  const handleAddModel = (rate: PriceBookModelRate): boolean => {
    const key = rate.match_key.trim().toLowerCase();
    if (key === "") {
      showErrorToast("Enter a match key.");
      return false;
    }
    const identity = `${key}:${rate.match_mode}`;
    if (modelKeys.has(identity)) {
      showErrorToast("That model rate already exists.");
      return false;
    }
    setModels((current) => [...current, { ...rate, match_key: key }]);
    return true;
  };

  const handleAddVM = (rate: PriceBookVMRate): boolean => {
    const key = rate.match_key.trim().toLowerCase();
    if (key === "") {
      showErrorToast("Enter a machine type.");
      return false;
    }
    const identity = `${key}:${rate.match_mode}`;
    if (vmKeys.has(identity)) {
      showErrorToast("That VM rate already exists.");
      return false;
    }
    setVMs((current) => [...current, { ...rate, match_key: key }]);
    return true;
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      const payload = await savePriceBooks(models, vms);
      applyCatalog(payload);
      showSuccessToast("Saved a new current price book.");
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to save price books");
    } finally {
      setSaving(false);
    }
  };

  const handleSync = async () => {
    setSyncing(true);
    try {
      const payload = await syncPriceBooks();
      applyCatalog(payload);
      showSuccessToast(`Updated ${payload.updated_count} model rates and added ${payload.added_count}.`);
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to update model rates");
    } finally {
      setSyncing(false);
    }
  };

  const handleActivate = async () => {
    if (!data) {
      return;
    }
    setActivating(true);
    try {
      const payload = await activatePriceBook(data.version);
      applyCatalog(payload);
      setActivateOpen(false);
      showSuccessToast("This version is now current.");
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to switch price book");
    } finally {
      setActivating(false);
    }
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center space-y-4 py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-b border-gray-500 dark:border-gray-400"></div>
        <Text className="text-gray-500 dark:text-gray-400">Loading price books...</Text>
      </div>
    );
  }

  if (loadFailed || !data) {
    return (
      <PriceBooksMessage
        message="We could not load price books."
        action={
          <Button className="mt-4" variant="outline" size="sm" onClick={() => void loadPriceBooks()}>
            Try again
          </Button>
        }
      />
    );
  }

  if (data.versions.length === 0) {
    return <PriceBooksMessage message="No price books in this installation." />;
  }

  return (
    <>
      <PriceBooksCatalog
        data={data}
        models={models}
        vms={vms}
        tab={tab}
        saving={saving}
        syncing={syncing}
        activating={activating}
        onTabChange={setTab}
        onVersionChange={(version) => void loadPriceBooks(version)}
        onModelChange={(index, patch) =>
          setModels((current) => current.map((rate, rateIndex) => (rateIndex === index ? { ...rate, ...patch } : rate)))
        }
        onVMChange={(index, micros) =>
          setVMs((current) =>
            current.map((rate, rateIndex) => (rateIndex === index ? { ...rate, micros_per_second: micros } : rate)),
          )
        }
        onAddModel={handleAddModel}
        onAddVM={handleAddVM}
        onSave={() => void handleSave()}
        onSync={() => void handleSync()}
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
            disabled={activating}
            onClick={() => void handleActivate()}
          >
            {activating ? "Switching version..." : "Use this version"}
          </Button>
          <Button variant="outline" onClick={() => setActivateOpen(false)}>
            Keep current version
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
