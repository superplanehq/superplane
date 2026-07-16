import type { ConfigurationField } from "@/api-client";
import { BookOpen, ExternalLink } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { PayloadPreview } from "@/ui/BuildingBlocksSidebar/PayloadPreview";

const DOCS_SURFACE_CLASS = "bg-slate-100 dark:bg-gray-800";
const DOCS_REFERENCE_CLASS = "bg-slate-100 dark:bg-transparent";
const CONFIG_TABLE_HEADER_CLASS = cn(
  "text-left py-1.5 px-2 border-b border-gray-200 font-medium text-gray-700 dark:border-gray-700 dark:text-gray-300",
  DOCS_SURFACE_CLASS,
);

function ConfigTable({ fields, showTopBorder }: { fields: ConfigurationField[]; showTopBorder?: boolean }) {
  return (
    <div
      className={cn("w-full px-4 pt-3 pb-3", showTopBorder && "border-t border-slate-950/15 dark:border-gray-800/70")}
    >
      <span className="text-[13px] font-medium text-gray-500 dark:text-gray-400">Configuration</span>
      <div className="overflow-x-auto mt-2">
        <table className="text-xs w-full border-collapse">
          <thead>
            <tr>
              <th className={CONFIG_TABLE_HEADER_CLASS}>Field</th>
              <th className={CONFIG_TABLE_HEADER_CLASS}>Type</th>
              <th className={CONFIG_TABLE_HEADER_CLASS}>Description</th>
            </tr>
          </thead>
          <tbody>
            {fields.map((field) => (
              <tr key={field.name}>
                <td className="py-1.5 px-2 border-b border-gray-100 font-mono text-gray-700 dark:border-gray-700/70 dark:text-gray-200">
                  {field.label || field.name}
                  {field.required && <span className="text-red-400 ml-0.5 dark:text-red-400">*</span>}
                </td>
                <td className="py-1.5 px-2 border-b border-gray-100 text-gray-500 dark:border-gray-700/70 dark:text-gray-400">
                  {field.type || "string"}
                </td>
                <td className="py-1.5 px-2 border-b border-gray-100 text-gray-500 dark:border-gray-700/70 dark:text-gray-400">
                  {field.description || "—"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function DocsReferenceSection({ documentationUrl }: { documentationUrl: string }) {
  return (
    <div className={cn("px-4 py-2.5", DOCS_REFERENCE_CLASS)}>
      <Button variant="outline" size="xs" asChild>
        <a href={documentationUrl} target="_blank" rel="noopener noreferrer">
          <BookOpen className="size-3" aria-hidden />
          Docs reference
          <ExternalLink className="size-3" aria-hidden />
        </a>
      </Button>
    </div>
  );
}

function DocsDescriptionSection({ description, showBottomBorder }: { description: string; showBottomBorder: boolean }) {
  return (
    <div
      className={cn(
        "w-full px-4 pt-3 pb-3",
        showBottomBorder && "border-b border-slate-950/15 dark:border-gray-800/70",
      )}
    >
      <span className="text-[13px] font-medium text-gray-500 dark:text-gray-400">Description</span>
      <p className="text-[13px] text-gray-800 mt-1 leading-relaxed dark:text-gray-100">{description}</p>
    </div>
  );
}

function DocsPayloadSection({
  examplePayload,
  payloadLabel,
  showBottomBorder,
}: {
  examplePayload: Record<string, unknown>;
  payloadLabel: string;
  showBottomBorder: boolean;
}) {
  return (
    <div className={cn("w-full px-2 py-2", showBottomBorder && "border-b border-slate-950/15 dark:border-gray-800/70")}>
      <div className="px-2">
        <PayloadPreview
          value={examplePayload}
          label={payloadLabel}
          dialogTitle={payloadLabel}
          maxHeight="max-h-64"
          showCopy
          labelSize="md"
        />
      </div>
    </div>
  );
}

interface DocsTabProps {
  description?: string;
  examplePayload?: Record<string, unknown>;
  payloadLabel?: string;
  /** Opens in a new tab; typically docs.superplane.com for this block. */
  documentationUrl?: string;
  configurationFields?: ConfigurationField[];
}

export function DocsTab({
  description,
  examplePayload,
  payloadLabel = "Example Output",
  documentationUrl,
  configurationFields = [],
}: DocsTabProps) {
  const hasPayload = examplePayload && Object.keys(examplePayload).length > 0;
  const hasFollowingContent = Boolean(hasPayload || configurationFields.length > 0);
  const hasContent = Boolean(description || hasPayload || configurationFields.length > 0 || documentationUrl);

  if (!hasContent) {
    return (
      <div className="flex flex-col items-center justify-center py-16 px-6 text-center">
        <p className="text-sm text-gray-500 dark:text-gray-400">No documentation available for this component.</p>
      </div>
    );
  }

  return (
    <div className="pb-8">
      {documentationUrl ? <DocsReferenceSection documentationUrl={documentationUrl} /> : null}
      {description ? <DocsDescriptionSection description={description} showBottomBorder={hasFollowingContent} /> : null}
      {hasPayload ? (
        <DocsPayloadSection
          examplePayload={examplePayload!}
          payloadLabel={payloadLabel}
          showBottomBorder={configurationFields.length > 0}
        />
      ) : null}
      {configurationFields.length > 0 ? (
        <ConfigTable
          fields={configurationFields}
          showTopBorder={!hasPayload && Boolean(description || documentationUrl)}
        />
      ) : null}
    </div>
  );
}
