import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { showErrorToast } from "@/lib/toast";
import { BookOpen } from "lucide-react";
import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { formatDate } from "./formatDate";
import { formatCentsPerMillionUsd, formatMatchMode, formatMicrosPerSecondUsdPerMinute } from "./priceBookFormat";

type PriceBookVersion = {
  version: string;
  effective_at: string;
  created_at: string;
};

type PriceBookModelRate = {
  match_key: string;
  match_mode: string;
  input_cents_per_million: number;
  output_cents_per_million: number;
  cache_read_cents_per_million: number;
  cache_write_cents_per_million: number;
  reasoning_cents_per_million: number;
};

type PriceBookVMRate = {
  match_key: string;
  match_mode: string;
  micros_per_second: number;
};

type PriceBooksResponse = {
  current_version: string;
  version: string;
  effective_at: string;
  created_at: string;
  versions: PriceBookVersion[];
  models: PriceBookModelRate[];
  vms: PriceBookVMRate[];
};

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

function ModelsTable({ rates }: { rates: PriceBookModelRate[] }) {
  if (rates.length === 0) {
    return <EmptyRatesMessage message="This version has no model rates." />;
  }

  return (
    <div className={tableWrapClass}>
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
          {rates.map((rate) => (
            <tr key={`${rate.match_key}:${rate.match_mode}`} className={rowClass}>
              <td className={`${bodyCellClass} font-mono text-xs`}>{rate.match_key}</td>
              <td className={bodyCellClass}>{formatMatchMode(rate.match_mode)}</td>
              <td className={numericCellClass}>{formatCentsPerMillionUsd(rate.input_cents_per_million)}</td>
              <td className={numericCellClass}>{formatCentsPerMillionUsd(rate.output_cents_per_million)}</td>
              <td className={numericCellClass}>{formatCentsPerMillionUsd(rate.cache_read_cents_per_million)}</td>
              <td className={numericCellClass}>{formatCentsPerMillionUsd(rate.cache_write_cents_per_million)}</td>
              <td className={numericCellClass}>{formatCentsPerMillionUsd(rate.reasoning_cents_per_million)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function VMsTable({ rates }: { rates: PriceBookVMRate[] }) {
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
          {rates.map((rate) => (
            <tr key={`${rate.match_key}:${rate.match_mode}`} className={rowClass}>
              <td className={`${bodyCellClass} font-mono text-xs`}>{rate.match_key}</td>
              <td className={`${numericCellClass} font-mono text-xs`}>{rate.micros_per_second}</td>
              <td className={numericCellClass}>{formatMicrosPerSecondUsdPerMinute(rate.micros_per_second)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function PriceBooksCatalog({
  data,
  tab,
  onTabChange,
  onVersionChange,
}: {
  data: PriceBooksResponse;
  tab: PriceBooksTab;
  onTabChange: (tab: PriceBooksTab) => void;
  onVersionChange: (version: string) => void;
}) {
  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <PriceBooksHeader />
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
        <TabsContent value="models" className="mt-3">
          <ModelsTable rates={data.models} />
        </TabsContent>
        <TabsContent value="vms" className="mt-3">
          <VMsTable rates={data.vms} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

export function PriceBooks() {
  const [data, setData] = useState<PriceBooksResponse | null>(null);
  const [tab, setTab] = useState<PriceBooksTab>("models");
  const [loading, setLoading] = useState(true);
  const [loadFailed, setLoadFailed] = useState(false);
  const loadAbort = useRef<AbortController | null>(null);
  const loadGeneration = useRef(0);

  const loadPriceBooks = useCallback(async (version?: string) => {
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
      const path = version ? `/admin/api/price-books?version=${encodeURIComponent(version)}` : "/admin/api/price-books";
      const response = await fetch(path, { credentials: "include", signal: controller.signal });
      if (!response.ok) {
        const text = await response.text();
        throw new Error(text.trim() || "Failed to load price books");
      }

      const payload: PriceBooksResponse = await response.json();
      if (generation !== loadGeneration.current) {
        return;
      }

      setData(payload);
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
  }, []);

  useEffect(() => {
    void loadPriceBooks();
    return () => {
      loadAbort.current?.abort();
    };
  }, [loadPriceBooks]);

  useReportPageReady(!loading);

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
    <PriceBooksCatalog
      data={data}
      tab={tab}
      onTabChange={setTab}
      onVersionChange={(version) => void loadPriceBooks(version)}
    />
  );
}
