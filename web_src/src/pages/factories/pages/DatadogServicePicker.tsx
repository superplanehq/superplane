import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { Check, Loader2 } from "lucide-react";
import { useMemo } from "react";

import { DATADOG_INTAKE_SETUP_COPY } from "./datadogIntakeSetupCopy";

export function DatadogServicePicker({
  services,
  serviceName,
  loading,
  error,
  onChange,
  onRetry,
  showNameField = true,
}: {
  services: Array<{ id?: string; name?: string }>;
  serviceName: string;
  loading: boolean;
  error: boolean;
  onChange: (name: string) => void;
  onRetry: () => void;
  showNameField?: boolean;
}) {
  const options = useMemo(() => servicesWithCurrent(services, serviceName), [services, serviceName]);
  const filtered = useMemo(() => {
    if (!showNameField) return options;
    const term = serviceName.trim().toLowerCase();
    if (!term) return options;
    const exactSelected = options.some((service) => (service.id ?? service.name ?? "").toLowerCase() === term);
    if (exactSelected) return options;
    return options.filter((service) => (service.name ?? service.id ?? "").toLowerCase().includes(term));
  }, [options, serviceName, showNameField]);

  return (
    <div className="space-y-3">
      {showNameField ? (
        <div className="space-y-2">
          <Label htmlFor="datadog-service-name">{DATADOG_INTAKE_SETUP_COPY.wizardServiceNameLabel}</Label>
          <Input
            id="datadog-service-name"
            value={serviceName}
            onChange={(event) => onChange(event.target.value)}
            placeholder={DATADOG_INTAKE_SETUP_COPY.wizardServiceNamePlaceholder}
            data-testid="datadog-service-name"
          />
        </div>
      ) : null}
      {loading ? (
        <p className="workspace-body-text flex items-center gap-2 text-muted-foreground">
          <Loader2 className="size-4 animate-spin" aria-hidden />
          {DATADOG_INTAKE_SETUP_COPY.wizardServicesLoading}
        </p>
      ) : null}
      {error ? (
        <div className="space-y-3">
          <p className="workspace-body-text text-destructive">{DATADOG_INTAKE_SETUP_COPY.wizardServicesError}</p>
          <Button type="button" variant="outline" size="sm" onClick={onRetry}>
            {DATADOG_INTAKE_SETUP_COPY.wizardRetry}
          </Button>
        </div>
      ) : null}
      {!loading && !error && options.length === 0 ? (
        <p className="workspace-body-text text-muted-foreground">{DATADOG_INTAKE_SETUP_COPY.wizardServicesEmpty}</p>
      ) : null}
      {!loading && !error && options.length > 0 ? (
        <div className="max-h-56 overflow-y-auto rounded-lg border border-border" role="listbox" aria-label="Services">
          {filtered.length === 0 ? (
            <p className="px-3 py-6 text-center text-[13px] text-muted-foreground">
              {DATADOG_INTAKE_SETUP_COPY.wizardNoMatchingServices}
            </p>
          ) : (
            <ul className="divide-y divide-border">
              {filtered.map((service) => {
                const id = service.id ?? service.name ?? "";
                const selected = id === serviceName.trim();
                return (
                  <li key={id}>
                    <Button
                      type="button"
                      variant="ghost"
                      role="option"
                      aria-selected={selected}
                      onClick={() => onChange(id)}
                      data-testid={`datadog-service-${id}`}
                      className={cn(
                        "h-auto w-full justify-start gap-3 rounded-none px-3 py-2.5 text-left text-[13px] font-medium",
                        selected ? "bg-accent/50" : "hover:bg-accent/30",
                      )}
                    >
                      <span className="min-w-0 flex-1 truncate">{service.name || id}</span>
                      {selected ? (
                        <Check className="size-3.5 shrink-0 text-foreground" strokeWidth={2.5} aria-hidden />
                      ) : null}
                    </Button>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      ) : null}
    </div>
  );
}

function servicesWithCurrent(
  services: Array<{ id?: string; name?: string }>,
  serviceName: string,
): Array<{ id?: string; name?: string }> {
  const current = serviceName.trim();
  if (!current) {
    return services;
  }
  if (services.some((service) => (service.id ?? service.name ?? "") === current)) {
    return services;
  }
  return [{ id: current, name: current }, ...services];
}
