import { useEffect, useId, useState } from "react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { billingCatalogRequest } from "./billingPlansApi";
import type { PolarProduct, PolarProductKind } from "./polarProductsApi";

const productCopy = {
  one_time: {
    help: "Choose a one-time USD product from Polar. You can save without a product and link it later.",
    empty: "No one-time USD products found. Create a product in Polar, then refresh the list.",
  },
  subscription: {
    help: "Choose a subscription product in USD from Polar. You can save without a product and link it later.",
    empty: "No subscription products in USD found. Create a product in Polar, then refresh the list.",
  },
};

export function PolarProductSelect({
  value,
  onChange,
  disabled,
  kind = "one_time",
  description,
}: {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  kind?: PolarProductKind;
  description?: string;
}) {
  const id = useId();
  const state = usePolarProducts(kind);
  const { products, loading } = state;
  const selected = products.find((product) => product.id === value);
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>Polar product</Label>
      <Select
        value={value || "none"}
        onValueChange={(next) => onChange(next === "none" ? "" : next)}
        disabled={disabled || loading}
      >
        <SelectTrigger id={id} className="w-full" aria-describedby={`${id}-help`}>
          <SelectValue>{selected?.name ?? (value ? "Saved product" : "No product")}</SelectValue>
        </SelectTrigger>
        <SelectContent position="popper">
          <SelectItem value="none">No product</SelectItem>
          {value && !selected && <SelectItem value={value}>Saved product · {value}</SelectItem>}
          {products.map((product) => (
            <SelectItem key={product.id} value={product.id} textValue={product.name}>
              <span className="flex flex-col items-start">
                <span>{product.name}</span>
                <code className="text-xs text-muted-foreground">{product.id}</code>
              </span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {value && (
        <p className="text-xs text-muted-foreground">
          Product ID: <code>{value}</code>
        </p>
      )}
      <p id={`${id}-help`} className="text-sm text-muted-foreground">
        {description ?? productCopy[kind].help}
      </p>
      <PolarProductStatus
        state={state}
        disabled={disabled}
        missing={!!value && !selected}
        emptyMessage={productCopy[kind].empty}
      />
    </div>
  );
}

function usePolarProducts(kind: PolarProductKind) {
  const [products, setProducts] = useState<PolarProduct[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    let canceled = false;
    setLoading(true);
    setError("");
    const path = kind === "subscription" ? "polar/products?kind=subscription" : "polar/products";
    void billingCatalogRequest<PolarProduct[]>(path)
      .then((next) => {
        if (!canceled) setProducts(next);
      })
      .catch((cause) => {
        if (!canceled) setError(cause instanceof Error ? cause.message : "Could not load Polar products. Try again.");
      })
      .finally(() => {
        if (!canceled) setLoading(false);
      });
    return () => {
      canceled = true;
    };
  }, [attempt, kind]);
  return { products, loading, error, refresh: () => setAttempt((current) => current + 1) };
}

function PolarProductStatus({
  state,
  disabled,
  missing,
  emptyMessage,
}: {
  state: ReturnType<typeof usePolarProducts>;
  disabled?: boolean;
  missing: boolean;
  emptyMessage: string;
}) {
  const { products, loading, error, refresh } = state;
  return (
    <>
      {loading && (
        <p role="status" className="text-sm text-muted-foreground">
          Loading Polar products...
        </p>
      )}
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
      {!loading && !error && products.length === 0 && <p className="text-sm text-muted-foreground">{emptyMessage}</p>}
      {!loading && !error && missing && (
        <p className="text-sm text-muted-foreground">The saved product is not in the available product list.</p>
      )}
      <Button type="button" size="sm" variant="outline" disabled={disabled || loading} onClick={refresh}>
        {error ? "Retry products" : "Refresh products"}
      </Button>
    </>
  );
}
