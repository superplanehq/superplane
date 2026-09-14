import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { BookOpen } from "lucide-react";
import { useEffect, useState } from "react";
import {
  centsToUsdInput,
  formatMatchMode,
  formatMicrosPerSecondUsdPerMinute,
  usdInputToCents,
} from "./priceBookFormat";
import type { PriceBookModelRate, PriceBookVMRate } from "./priceBooksApi";

export const tableWrapClass =
  "bg-white rounded-md shadow-sm outline outline-slate-950/10 overflow-hidden dark:bg-gray-900 dark:outline-gray-700/70";
const headerCellClass = "text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400";
const numericHeaderCellClass = `${headerCellClass} text-right`;
const bodyCellClass = "px-4 py-2.5 text-gray-700 dark:text-gray-300";
const numericCellClass = `${bodyCellClass} text-right tabular-nums`;
const rowClass = "border-b border-slate-50 last:border-0 dark:border-gray-800/70";

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

export function ModelsTable({
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

function MachineTypeInput({
  id,
  matchKey,
  disabled,
  onCommit,
}: {
  id: string;
  matchKey: string;
  disabled: boolean;
  onCommit: (matchKey: string) => boolean;
}) {
  const [text, setText] = useState(matchKey);
  useEffect(() => {
    setText(matchKey);
  }, [matchKey]);

  return (
    <Input
      id={id}
      className="h-8 font-mono text-xs"
      disabled={disabled}
      value={text}
      onChange={(event) => setText(event.target.value)}
      onBlur={() => {
        if (!onCommit(text)) {
          setText(matchKey);
        }
      }}
    />
  );
}

export function VMsTable({
  rates,
  editable,
  onChange,
  onRemove,
}: {
  rates: PriceBookVMRate[];
  editable: boolean;
  onChange: (index: number, patch: Partial<PriceBookVMRate>) => boolean;
  onRemove: (index: number) => void;
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
            {editable && <th className={headerCellClass} />}
          </tr>
        </thead>
        <tbody>
          {rates.map((rate, index) => (
            <tr key={index} className={rowClass}>
              <td className={`${bodyCellClass} font-mono text-xs`}>
                {editable ? (
                  <MachineTypeInput
                    id={`price-book-vm-${index}-match-key`}
                    matchKey={rate.match_key}
                    disabled={!editable}
                    onCommit={(matchKey) => onChange(index, { match_key: matchKey })}
                  />
                ) : (
                  rate.match_key
                )}
              </td>
              <td className={numericCellClass}>
                {editable ? (
                  <Input
                    id={`price-book-vm-${index}-micros`}
                    type="number"
                    min="0"
                    step="1"
                    className="h-8 text-right font-mono text-xs tabular-nums"
                    value={rate.micros_per_second}
                    onChange={(event) => {
                      const parsed = Number.parseInt(event.target.value, 10);
                      onChange(index, {
                        micros_per_second: Number.isFinite(parsed) && parsed >= 0 ? parsed : 0,
                      });
                    }}
                  />
                ) : (
                  <span className="font-mono text-xs">{rate.micros_per_second}</span>
                )}
              </td>
              <td className={numericCellClass}>{formatMicrosPerSecondUsdPerMinute(rate.micros_per_second)}</td>
              {editable && (
                <td className={bodyCellClass}>
                  <Button type="button" variant="ghost" size="sm" onClick={() => onRemove(index)}>
                    Remove
                  </Button>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
