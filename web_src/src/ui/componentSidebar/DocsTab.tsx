import type { ConfigurationField } from "@/api-client";
import { BookOpen, ExternalLink } from "lucide-react";
import { PayloadPreview } from "@/ui/BuildingBlocksSidebar/PayloadPreview";

function ConfigTable({ fields }: { fields: ConfigurationField[] }) {
  return (
    <div className="w-full px-4 pt-3 pb-3 border-t border-gray-200">
      <span className="text-[13px] font-medium text-gray-500">Configuration</span>
      <div className="overflow-x-auto mt-2">
        <table className="text-xs w-full border-collapse">
          <thead>
            <tr>
              <th className="text-left py-1.5 px-2 border-b border-gray-200 font-medium text-gray-700 bg-gray-50">
                Field
              </th>
              <th className="text-left py-1.5 px-2 border-b border-gray-200 font-medium text-gray-700 bg-gray-50">
                Type
              </th>
              <th className="text-left py-1.5 px-2 border-b border-gray-200 font-medium text-gray-700 bg-gray-50">
                Description
              </th>
            </tr>
          </thead>
          <tbody>
            {fields.map((field) => (
              <tr key={field.name}>
                <td className="py-1.5 px-2 border-b border-gray-100 font-mono text-gray-700">
                  {field.label || field.name}
                  {field.required && <span className="text-red-400 ml-0.5">*</span>}
                </td>
                <td className="py-1.5 px-2 border-b border-gray-100 text-gray-500">{field.type || "string"}</td>
                <td className="py-1.5 px-2 border-b border-gray-100 text-gray-500">{field.description || "—"}</td>
              </tr>
            ))}
          </tbody>
        </table>
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

  if (!description && !hasPayload && configurationFields.length === 0 && !documentationUrl) {
    return (
      <div className="flex flex-col items-center justify-center py-16 px-6 text-center">
        <p className="text-sm text-gray-500">No documentation available for this component.</p>
      </div>
    );
  }

  return (
    <div className="pb-8">
      {documentationUrl && (
        <div className="border-b border-gray-200 dark:border-gray-700 px-4 py-2.5">
          <a
            href={documentationUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1.5 text-[11px] text-gray-500 dark:text-gray-400 hover:text-primary transition-colors"
          >
            <BookOpen size={12} className="shrink-0" aria-hidden />
            <span>Docs reference</span>
            <ExternalLink size={10} className="shrink-0" aria-hidden />
          </a>
        </div>
      )}
      {description && (
        <div className="w-full px-4 pt-3 pb-3">
          <span className="text-[13px] font-medium text-gray-500">Description</span>
          <p className="text-[13px] text-gray-800 mt-1 leading-relaxed">{description}</p>
        </div>
      )}

      {hasPayload && (
        <div className="w-full px-2 py-2 border-t border-gray-200">
          <div className="px-2">
            <PayloadPreview
              value={examplePayload!}
              label={payloadLabel}
              dialogTitle={payloadLabel}
              maxHeight="max-h-64"
              showCopy
              labelSize="md"
            />
          </div>
        </div>
      )}

      {configurationFields.length > 0 && <ConfigTable fields={configurationFields} />}
    </div>
  );
}
