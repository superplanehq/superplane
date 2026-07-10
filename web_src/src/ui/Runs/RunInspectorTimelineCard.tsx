import JsonView from "@uiw/react-json-view";
import * as AccordionPrimitive from "@radix-ui/react-accordion";
import { Check, ChevronDown, Copy, Maximize2 } from "lucide-react";
import { useMemo, useState, type CSSProperties, type ReactNode } from "react";
import { cn } from "@/lib/utils";
import { jsonViewClassName } from "@/lib/jsonViewTheme";
import { AccordionContent, AccordionItem } from "@/ui/accordion";
import { FullscreenContentDialog } from "@/ui/FullscreenContentDialog";
import { HeaderIconButton } from "@/ui/HeaderIconButton";
import { escapeJsonStringValue } from "./runInspectorJson";
import type { InternalAccordionKey, StatusPill } from "./RunInspectorTimelineTypes";

export function TimelineAccordionCard({
  value,
  status,
  title,
  sourceName,
  sourceTrailing,
  trailing,
  actionPayload,
  jsonViewStyle,
  errorOutputNodeId,
  children,
}: {
  value: InternalAccordionKey;
  status: StatusPill;
  title: string;
  sourceName?: string;
  sourceTrailing?: ReactNode;
  trailing?: ReactNode;
  actionPayload?: unknown;
  jsonViewStyle: CSSProperties;
  errorOutputNodeId?: string;
  children: ReactNode;
}) {
  const [modalOpen, setModalOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [modalCopied, setModalCopied] = useState(false);
  const canUsePayloadActions = isDisplayablePayload(actionPayload);
  const payloadString = useMemo(() => JSON.stringify(actionPayload ?? {}, null, 2), [actionPayload]);

  const copyPayload = (markCopied: (value: boolean) => void) => {
    void navigator.clipboard?.writeText(payloadString).catch(() => {});
    markCopied(true);
    setTimeout(() => markCopied(false), 1500);
  };

  return (
    <>
      <AccordionItem
        value={value}
        className="overflow-hidden rounded border border-slate-200 bg-white dark:border-gray-800 dark:bg-gray-900"
      >
        <AccordionPrimitive.Header className="flex items-center gap-1 border-b border-slate-200 bg-slate-50 py-1.5 pr-2 dark:border-gray-800 dark:bg-gray-900">
          <AccordionPrimitive.Trigger className="flex min-w-0 items-center gap-1.5 px-3 text-left hover:no-underline">
            <EventStatusPill {...status} />
            <span className="shrink-0 text-[11px] font-semibold uppercase tracking-wide text-slate-400 dark:text-gray-500">
              {title}
            </span>
            {sourceName ? (
              <span className="min-w-0 truncate text-[12px] font-medium text-slate-600 dark:text-gray-200">
                {sourceName}
              </span>
            ) : null}
          </AccordionPrimitive.Trigger>
          {sourceTrailing}
          <span className="ml-auto flex shrink-0 items-center gap-0.5 text-xs text-slate-700 dark:text-gray-200">
            {trailing ? <span className="pr-1">{trailing}</span> : null}
            {canUsePayloadActions ? (
              <>
                <HeaderIconButton
                  label={copied ? "Copied" : "Copy"}
                  icon={copied ? <Check className="h-3.5 w-3.5 text-emerald-600" /> : <Copy className="h-3.5 w-3.5" />}
                  onClick={() => copyPayload(setCopied)}
                />
                <HeaderIconButton
                  label="Open fullscreen"
                  icon={<Maximize2 className="h-3.5 w-3.5" />}
                  onClick={() => setModalOpen(true)}
                />
              </>
            ) : null}
          </span>
          <AccordionPrimitive.Trigger
            aria-label="Toggle section"
            className="flex h-6 w-6 shrink-0 items-center justify-center rounded text-slate-400 transition-colors hover:bg-slate-200 hover:text-slate-700 data-[state=open]:text-slate-600 dark:hover:bg-gray-800 dark:hover:text-gray-100 dark:data-[state=open]:text-gray-300 [&[data-state=open]>svg]:rotate-180"
          >
            <ChevronDown className="h-4 w-4 transition-transform duration-200" />
          </AccordionPrimitive.Trigger>
        </AccordionPrimitive.Header>
        <AccordionContent className="px-3 py-2.5">
          <div data-run-error-output-node-id={errorOutputNodeId}>{children}</div>
        </AccordionContent>
      </AccordionItem>

      <FullscreenContentDialog
        open={modalOpen}
        onOpenChange={setModalOpen}
        title={title}
        headerActions={
          <HeaderIconButton
            label={modalCopied ? "Copied" : "Copy"}
            icon={modalCopied ? <Check className="h-3.5 w-3.5 text-emerald-600" /> : <Copy className="h-3.5 w-3.5" />}
            onClick={() => copyPayload(setModalCopied)}
          />
        }
      >
        <JsonPayload value={actionPayload} jsonViewStyle={jsonViewStyle} collapsed={false} />
      </FullscreenContentDialog>
    </>
  );
}

export function JsonPayload({
  value,
  jsonViewStyle,
  collapsed = 2,
}: {
  value: unknown;
  jsonViewStyle: CSSProperties;
  collapsed?: boolean | number;
}) {
  return (
    <JsonView
      value={(value ?? {}) as object}
      collapsed={collapsed}
      style={jsonViewStyle}
      className={jsonViewClassName}
      displayObjectSize={false}
      enableClipboard={false}
    >
      <JsonView.String
        render={({ children, ...props }, { type, value: stringValue }) => {
          if (type !== "value") return undefined;

          const displayValue = typeof children === "string" ? children : String(stringValue ?? "");

          return (
            <>
              <span aria-hidden className={props.className}>
                &quot;
              </span>
              <span {...props}>{escapeJsonStringValue(displayValue)}</span>
              <span aria-hidden className={props.className}>
                &quot;
              </span>
            </>
          );
        }}
      />
    </JsonView>
  );
}

export function DetailBox({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="overflow-hidden rounded border border-slate-200 bg-white dark:border-gray-800 dark:bg-gray-900">
      <div className="flex items-center justify-between gap-2 border-b border-slate-200 bg-slate-50 px-3 py-1.5 dark:border-gray-800 dark:bg-gray-900">
        <span className="text-[11px] font-semibold uppercase tracking-wide text-slate-500 dark:text-gray-400">
          {title}
        </span>
      </div>
      <div className="px-3 py-2.5">{children}</div>
    </div>
  );
}

export function ErrorOutputCard({ nodeId, message }: { nodeId: string; message?: string }) {
  return (
    <div
      className="overflow-hidden rounded border border-red-200 bg-red-50 text-red-700 dark:border-red-900/70 dark:bg-red-950/30 dark:text-red-300"
      data-run-error-output-node-id={nodeId}
    >
      <div className="flex items-center justify-between gap-1.5 border-b border-red-200 px-3 py-1.5 dark:border-red-900/70">
        <span className="truncate text-[11px] font-semibold uppercase tracking-wide text-red-600">
          Error - Output not emitted
        </span>
      </div>
      <div className="px-3 py-2.5 text-[13px]">
        <span className="min-w-0 break-all font-medium">{message}</span>
      </div>
    </div>
  );
}

export function EmptySectionText({ children }: { children: ReactNode }) {
  return <p className="text-sm text-slate-500 dark:text-gray-400">{children}</p>;
}

function EventStatusPill({ dotClassName, label, tone = "default" }: StatusPill) {
  return (
    <span
      className={cn(
        "flex shrink-0 items-center gap-1.5 rounded-full bg-white px-2 py-0.5 ring-1 dark:bg-gray-950",
        tone === "error" ? "ring-red-200 dark:ring-red-900/70" : "ring-slate-200 dark:ring-gray-800",
      )}
    >
      <span className={cn("h-2 w-2 shrink-0 rounded-full", dotClassName)} />
      <span
        className={cn(
          "text-[11px] font-medium capitalize",
          tone === "error" ? "text-red-600" : "text-slate-700 dark:text-gray-200",
        )}
      >
        {label}
      </span>
    </span>
  );
}

function isDisplayablePayload(value: unknown): boolean {
  if (typeof value === "string") return value.length > 0;
  return !!value && typeof value === "object";
}
