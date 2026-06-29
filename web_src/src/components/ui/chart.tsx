import * as React from "react";
import * as RechartsPrimitive from "recharts";
import type { TooltipPayloadEntry, TooltipValueType } from "recharts";

import { cn } from "@/lib/utils";

import { toChartColorVarName } from "./chartColorVarName";

// Format: { THEME_NAME: CSS_SELECTOR }
const THEMES = { light: "", dark: ".dark" } as const;

const INITIAL_DIMENSION = { width: 320, height: 200 } as const;
type TooltipNameType = number | string;
type TooltipContentProps = RechartsPrimitive.DefaultTooltipContentProps<TooltipValueType, TooltipNameType>;
type TooltipPayloadItem = TooltipPayloadEntry<TooltipValueType, TooltipNameType>;
type TooltipPayloadItems = readonly TooltipPayloadItem[];

export type ChartConfig = Record<
  string,
  {
    label?: React.ReactNode;
    icon?: React.ComponentType;
  } & ({ color?: string; theme?: never } | { color?: never; theme: Record<keyof typeof THEMES, string> })
>;

type ChartContextProps = {
  config: ChartConfig;
};

const ChartContext = React.createContext<ChartContextProps | null>(null);

function useChart() {
  const context = React.useContext(ChartContext);

  if (!context) {
    throw new Error("useChart must be used within a <ChartContainer />");
  }

  return context;
}

function ChartContainer({
  id,
  className,
  children,
  config,
  initialDimension = INITIAL_DIMENSION,
  ...props
}: React.ComponentProps<"div"> & {
  config: ChartConfig;
  children: React.ComponentProps<typeof RechartsPrimitive.ResponsiveContainer>["children"];
  initialDimension?: {
    width: number;
    height: number;
  };
}) {
  const uniqueId = React.useId();
  const chartId = `chart-${id ?? uniqueId.replace(/:/g, "")}`;

  return (
    <ChartContext.Provider value={{ config }}>
      <div
        data-slot="chart"
        data-chart={chartId}
        className={cn(
          "flex aspect-video justify-center text-xs [&_.recharts-cartesian-axis-tick_text]:fill-muted-foreground [&_.recharts-cartesian-grid_line[stroke='#ccc']]:stroke-border/50 [&_.recharts-curve.recharts-tooltip-cursor]:stroke-border [&_.recharts-dot[stroke='#fff']]:stroke-transparent [&_.recharts-layer]:outline-hidden [&_.recharts-polar-grid_[stroke='#ccc']]:stroke-border [&_.recharts-radial-bar-background-sector]:fill-muted [&_.recharts-rectangle.recharts-tooltip-cursor]:fill-muted [&_.recharts-reference-line_[stroke='#ccc']]:stroke-border [&_.recharts-sector]:outline-hidden [&_.recharts-sector[stroke='#fff']]:stroke-transparent [&_.recharts-surface]:outline-hidden",
          className,
        )}
        {...props}
      >
        <ChartStyle id={chartId} config={config} />
        <RechartsPrimitive.ResponsiveContainer initialDimension={initialDimension}>
          {children}
        </RechartsPrimitive.ResponsiveContainer>
      </div>
    </ChartContext.Provider>
  );
}

const ChartStyle = ({ id, config }: { id: string; config: ChartConfig }) => {
  const colorConfig = Object.entries(config).filter(([, config]) => config.theme ?? config.color);

  if (!colorConfig.length) {
    return null;
  }

  return (
    <style
      dangerouslySetInnerHTML={{
        __html: Object.entries(THEMES)
          .map(
            ([theme, prefix]) => `
${prefix} [data-chart=${id}] {
${colorConfig
  .map(([key, itemConfig]) => {
    const color = itemConfig.theme?.[theme as keyof typeof itemConfig.theme] ?? itemConfig.color;
    return color ? `  --color-${toChartColorVarName(key)}: ${color};` : null;
  })
  .join("\n")}
}
`,
          )
          .join("\n"),
      }}
    />
  );
};

const ChartTooltip = RechartsPrimitive.Tooltip;

function resolveTooltipLabelValue(
  config: ChartConfig,
  payload: TooltipPayloadItems,
  labelKey?: string,
  label?: React.ReactNode,
) {
  const [item] = payload;
  const key = `${labelKey ?? item?.dataKey ?? item?.name ?? "value"}`;
  const itemConfig = getPayloadConfigFromPayload(config, item, key);
  return !labelKey && typeof label === "string" ? (config[label]?.label ?? label) : itemConfig?.label;
}

function renderTooltipLabel(opts: {
  hideLabel?: boolean;
  payload?: TooltipPayloadItems;
  labelKey?: string;
  label?: React.ReactNode;
  labelFormatter?: TooltipContentProps["labelFormatter"];
  labelClassName?: string;
  config: ChartConfig;
}) {
  const { hideLabel, payload, labelKey, label, labelFormatter, labelClassName, config } = opts;
  if (hideLabel || !payload?.length) return null;

  const value = resolveTooltipLabelValue(config, payload, labelKey, label);

  if (labelFormatter) {
    return <div className={cn("font-medium", labelClassName)}>{labelFormatter(value, payload)}</div>;
  }
  if (!value) return null;
  return <div className={cn("font-medium", labelClassName)}>{value}</div>;
}

function ChartTooltipContent({
  active,
  payload,
  className,
  indicator = "dot",
  hideLabel = false,
  hideIndicator = false,
  label,
  labelFormatter,
  labelClassName,
  formatter,
  color,
  nameKey,
  labelKey,
}: React.ComponentProps<typeof RechartsPrimitive.Tooltip> &
  React.ComponentProps<"div"> & {
    hideLabel?: boolean;
    hideIndicator?: boolean;
    indicator?: "line" | "dot" | "dashed";
    nameKey?: string;
    labelKey?: string;
  } & Omit<RechartsPrimitive.DefaultTooltipContentProps<TooltipValueType, TooltipNameType>, "accessibilityLayer">) {
  const { config } = useChart();

  const tooltipLabel = React.useMemo(
    () =>
      renderTooltipLabel({
        hideLabel,
        payload,
        labelKey,
        label,
        labelFormatter,
        labelClassName,
        config,
      }),
    [label, labelFormatter, payload, hideLabel, labelClassName, config, labelKey],
  );

  if (!active || !payload?.length) {
    return null;
  }

  const nestLabel = payload.length === 1 && indicator !== "dot";

  return (
    <div
      className={cn(
        "grid min-w-[8rem] items-start gap-1.5 rounded-lg border border-border/50 bg-background px-2.5 py-1.5 text-xs shadow-xl",
        className,
      )}
    >
      {!nestLabel ? tooltipLabel : null}
      <div className="grid gap-1.5">
        {payload
          .filter((item) => item.type !== "none")
          .map((item, index) => (
            <TooltipPayloadRow
              key={index}
              item={item}
              index={index}
              config={config}
              nameKey={nameKey}
              color={color}
              indicator={indicator}
              hideIndicator={hideIndicator}
              nestLabel={nestLabel}
              tooltipLabel={tooltipLabel}
              formatter={formatter}
              tooltipPayload={payload}
            />
          ))}
      </div>
    </div>
  );
}

const ChartLegend = RechartsPrimitive.Legend;

function TooltipIndicator({
  itemConfig,
  hideIndicator,
  indicator,
  nestLabel,
  indicatorColor,
}: {
  itemConfig?: { icon?: React.ComponentType; label?: React.ReactNode };
  hideIndicator?: boolean;
  indicator?: string;
  nestLabel: boolean;
  indicatorColor?: string;
}) {
  if (itemConfig?.icon) return <itemConfig.icon />;
  if (hideIndicator) return null;
  return (
    <div
      className={cn("shrink-0 rounded-[2px] border-(--color-border) bg-(--color-bg)", {
        "h-2.5 w-2.5": indicator === "dot",
        "w-1": indicator === "line",
        "w-0 border-[1.5px] border-dashed bg-transparent": indicator === "dashed",
        "my-0.5": nestLabel && indicator === "dashed",
      })}
      style={{ "--color-bg": indicatorColor, "--color-border": indicatorColor } as React.CSSProperties}
    />
  );
}

function resolvePayloadKey(nameKey?: string, item?: { name?: TooltipNameType; dataKey?: unknown }) {
  return `${nameKey ?? item?.name ?? item?.dataKey ?? "value"}`;
}

function resolveIndicatorColor(color?: string, item?: { payload?: { fill?: string }; color?: string }) {
  return color ?? item?.payload?.fill ?? item?.color;
}

function formatPayloadValue(value: TooltipValueType) {
  return typeof value === "number" ? value.toLocaleString() : String(value);
}

function TooltipPayloadRow({
  item,
  index,
  config,
  nameKey,
  color,
  indicator,
  hideIndicator,
  nestLabel,
  tooltipLabel,
  formatter,
  tooltipPayload,
}: {
  item: TooltipPayloadItem;
  index: number;
  config: ChartConfig;
  nameKey?: string;
  color?: string;
  indicator?: string;
  hideIndicator?: boolean;
  nestLabel: boolean;
  tooltipLabel: React.ReactNode;
  formatter?: TooltipContentProps["formatter"];
  tooltipPayload: TooltipPayloadItems;
}) {
  const key = resolvePayloadKey(nameKey, item);
  const itemConfig = getPayloadConfigFromPayload(config, item, key);
  const indicatorColor = resolveIndicatorColor(color, item);

  return (
    <div
      className={cn(
        "flex w-full flex-wrap items-stretch gap-2 [&>svg]:h-2.5 [&>svg]:w-2.5 [&>svg]:text-muted-foreground",
        indicator === "dot" && "items-center",
      )}
    >
      {formatter && item?.value !== undefined && item.name ? (
        formatter(item.value, item.name, item, index, tooltipPayload)
      ) : (
        <>
          <TooltipIndicator
            itemConfig={itemConfig}
            hideIndicator={hideIndicator}
            indicator={indicator}
            nestLabel={nestLabel}
            indicatorColor={indicatorColor}
          />
          <div className={cn("flex flex-1 justify-between leading-none", nestLabel ? "items-end" : "items-center")}>
            <div className="grid gap-1.5">
              {nestLabel ? tooltipLabel : null}
              <span className="text-muted-foreground">{itemConfig?.label ?? item.name}</span>
            </div>
            {item.value != null && (
              <span className="font-mono font-medium text-foreground tabular-nums">
                {formatPayloadValue(item.value)}
              </span>
            )}
          </div>
        </>
      )}
    </div>
  );
}

function ChartLegendContent({
  className,
  hideIcon = false,
  payload,
  verticalAlign = "bottom",
  nameKey,
}: React.ComponentProps<"div"> & {
  hideIcon?: boolean;
  nameKey?: string;
} & RechartsPrimitive.DefaultLegendContentProps) {
  const { config } = useChart();

  if (!payload?.length) {
    return null;
  }

  return (
    <div className={cn("flex items-center justify-center gap-4", verticalAlign === "top" ? "pb-3" : "pt-3", className)}>
      {payload
        .filter((item) => item.type !== "none")
        .map((item, index) => {
          const key = `${nameKey ?? item.dataKey ?? "value"}`;
          const itemConfig = getPayloadConfigFromPayload(config, item, key);

          return (
            <div
              key={index}
              className={cn("flex items-center gap-1.5 [&>svg]:h-3 [&>svg]:w-3 [&>svg]:text-muted-foreground")}
            >
              {itemConfig?.icon && !hideIcon ? (
                <itemConfig.icon />
              ) : (
                <div
                  className="h-2 w-2 shrink-0 rounded-[2px]"
                  style={{
                    backgroundColor: itemConfig?.color ?? item.color,
                  }}
                />
              )}
              {itemConfig?.label}
            </div>
          );
        })}
    </div>
  );
}

// Helper to extract item config from a payload.
function getPayloadConfigFromPayload(config: ChartConfig, payload: unknown, key: string) {
  if (typeof payload !== "object" || payload === null) {
    return undefined;
  }

  const payloadPayload =
    "payload" in payload && typeof payload.payload === "object" && payload.payload !== null
      ? payload.payload
      : undefined;

  let configLabelKey: string = key;

  if (key in payload && typeof payload[key as keyof typeof payload] === "string") {
    configLabelKey = payload[key as keyof typeof payload] as string;
  } else if (
    payloadPayload &&
    key in payloadPayload &&
    typeof payloadPayload[key as keyof typeof payloadPayload] === "string"
  ) {
    configLabelKey = payloadPayload[key as keyof typeof payloadPayload] as string;
  }

  return configLabelKey in config ? config[configLabelKey] : config[key];
}

export { ChartContainer, ChartTooltip, ChartTooltipContent, ChartLegend, ChartLegendContent, ChartStyle };
