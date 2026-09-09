import { Text } from "@/components/Text/text";
import { Input, InputGroup } from "@/components/Input/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Timestamp } from "@/components/Timestamp";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { formatCentsPerMillion, formatMicrosPerHour } from "@/lib/priceBookRates";
import { BookOpen, Search } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";

type PriceBookRate = {
  usage_kind: string;
  match_key: string;
  match_mode: string;
  input_cents_per_million: number;
  output_cents_per_million: number;
  cache_read_cents_per_million: number;
  cache_write_cents_per_million: number;
  reasoning_cents_per_million: number;
  micros_per_second: number;
};

type PriceBookResponse = {
  version: string;
  previous_version?: string;
  effective_at?: string | null;
  model_rate_count: number;
  compute_rate_count: number;
  sources?: string[];
  rates: PriceBookRate[];
};

const loadErrorMessage = "SuperPlane could not load the price book.";
const scanErrorMessage = "SuperPlane could not scan provider prices. Try again.";

function PriceBookPage() {
  const [book, setBook] = useState<PriceBookResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [scanning, setScanning] = useState(false);
  const [search, setSearch] = useState("");

  const applyBook = useCallback((data: PriceBookResponse) => {
    setBook({
      ...data,
      rates: data.rates ?? [],
    });
  }, []);

  const loadBook = useCallback(async () => {
    try {
      const response = await fetch("/admin/api/installation/price-book", { credentials: "include" });
      if (!response.ok) {
        const text = await response.text();
        throw new Error(text.trim() || loadErrorMessage);
      }
      applyBook((await response.json()) as PriceBookResponse);
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : loadErrorMessage);
    } finally {
      setLoading(false);
    }
  }, [applyBook]);

  const scanBook = useCallback(async () => {
    setScanning(true);
    try {
      const response = await fetch("/admin/api/installation/price-book/scan", {
        method: "POST",
        credentials: "include",
      });
      if (!response.ok) {
        const text = await response.text();
        throw new Error(text.trim() || scanErrorMessage);
      }
      const data = (await response.json()) as PriceBookResponse;
      applyBook(data);
      showSuccessToast(`Price book updated to version ${data.version}.`);
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : scanErrorMessage);
    } finally {
      setScanning(false);
    }
  }, [applyBook]);

  useEffect(() => {
    void loadBook();
  }, [loadBook]);

  useReportPageReady(!loading);

  const filteredRates = useMemo(() => {
    const query = search.trim().toLowerCase();
    const rates = book?.rates ?? [];
    if (query === "") {
      return rates;
    }
    return rates.filter((rate) => {
      return (
        rate.match_key.toLowerCase().includes(query) ||
        rate.usage_kind.toLowerCase().includes(query) ||
        rate.match_mode.toLowerCase().includes(query)
      );
    });
  }, [book, search]);

  if (loading && book == null) {
    return (
      <div className="flex flex-col items-center space-y-4 py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-b border-gray-500 dark:border-gray-400"></div>
        <Text className="text-gray-500 dark:text-gray-400">Loading price book...</Text>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">Price book</h1>
          <Text className="mt-1 max-w-2xl text-sm text-gray-500 dark:text-gray-400">
            Scan OpenRouter model prices and SuperPlane runner VM rates. SuperPlane writes a new catalog version and
            loads it without a restart.
          </Text>
        </div>
        <Button type="button" data-testid="admin-price-book-scan" onClick={() => void scanBook()} disabled={scanning}>
          {scanning ? "Scanning..." : "Scan and update"}
        </Button>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        <SummaryCard label="Version" value={book?.version || "—"} testId="admin-price-book-version" />
        <SummaryCard
          label="Model rates"
          value={String(book?.model_rate_count ?? 0)}
          testId="admin-price-book-model-count"
        />
        <SummaryCard
          label="VM rates"
          value={String(book?.compute_rate_count ?? 0)}
          testId="admin-price-book-compute-count"
        />
      </div>

      <Text className="text-sm text-gray-500 dark:text-gray-400">
        Effective{" "}
        <Timestamp
          date={book?.effective_at ?? undefined}
          className="text-gray-700 dark:text-gray-300"
          fallback={<span>—</span>}
        />
      </Text>

      <div className="max-w-md">
        <Label htmlFor="admin-price-book-search">Search rates</Label>
        <InputGroup className="relative mt-1">
          <Search
            size={14}
            className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
          />
          <Input
            id="admin-price-book-search"
            data-testid="admin-price-book-search"
            type="search"
            className="pl-9"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Model or machine type"
          />
        </InputGroup>
      </div>

      {filteredRates.length === 0 ? (
        <div className="rounded-xl border border-slate-200 bg-white p-8 text-center shadow-sm dark:border-gray-700 dark:bg-gray-900">
          <BookOpen size={24} className="mx-auto text-gray-400 dark:text-gray-500" />
          <Text className="mt-3 text-sm text-gray-600 dark:text-gray-400">
            {search.trim() === "" ? "No price book rates are stored yet." : "No rates match this search."}
          </Text>
        </div>
      ) : (
        <PriceBookRatesTable rates={filteredRates} />
      )}
    </div>
  );
}

function SummaryCard({ label, value, testId }: { label: string; value: string; testId: string }) {
  return (
    <div className="rounded-md bg-white px-4 py-3 shadow-sm outline outline-slate-950/10 dark:bg-gray-900 dark:outline-gray-700/70">
      <p className="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{label}</p>
      <p className="mt-1 font-mono text-sm text-gray-900 dark:text-gray-100" data-testid={testId}>
        {value}
      </p>
    </div>
  );
}

function PriceBookRatesTable({ rates }: { rates: PriceBookRate[] }) {
  return (
    <div className="overflow-hidden rounded-md bg-white shadow-sm outline outline-slate-950/10 dark:bg-gray-900 dark:outline-gray-700/70">
      <table className="w-full text-sm" data-testid="admin-price-book-rates">
        <thead>
          <tr className="border-b border-slate-100 dark:border-gray-700/70">
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Kind</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Match</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Mode</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Price</th>
          </tr>
        </thead>
        <tbody>
          {rates.map((rate) => (
            <tr
              key={`${rate.usage_kind}:${rate.match_mode}:${rate.match_key}`}
              className="border-b border-slate-50 last:border-0 dark:border-gray-800"
            >
              <td className="px-4 py-2.5 text-gray-700 dark:text-gray-300">{rate.usage_kind}</td>
              <td className="px-4 py-2.5 font-mono text-gray-900 dark:text-gray-100">{rate.match_key}</td>
              <td className="px-4 py-2.5 text-gray-700 dark:text-gray-300">{rate.match_mode}</td>
              <td className="px-4 py-2.5 text-gray-700 dark:text-gray-300">{ratePriceLabel(rate)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ratePriceLabel(rate: PriceBookRate): string {
  if (rate.usage_kind === "compute") {
    return formatMicrosPerHour(rate.micros_per_second);
  }
  return `in ${formatCentsPerMillion(rate.input_cents_per_million)}, out ${formatCentsPerMillion(rate.output_cents_per_million)}`;
}

export default PriceBookPage;
