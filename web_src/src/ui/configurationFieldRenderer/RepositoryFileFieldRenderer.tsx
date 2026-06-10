import React from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { canvasesListCanvasRepositoryFiles } from "@/api-client/sdk.gen";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { toTestId } from "@/lib/testID";
import type { FieldRendererProps } from "./types";

export const RepositoryFileFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange }) => {
  const { appId } = useParams<{ appId: string }>();

  const {
    data: files = [],
    isLoading,
    isError,
  } = useQuery({
    queryKey: ["repository-files", appId],
    queryFn: async () => {
      if (!appId) return [];
      const response = await canvasesListCanvasRepositoryFiles(
        withOrganizationHeader({
          path: { canvasId: appId },
        }),
      );
      return response.data?.files?.map((f) => f.path ?? "").filter(Boolean) ?? [];
    },
    enabled: !!appId,
    staleTime: 30 * 1000,
  });

  const testId = field.name ? toTestId(`field-${field.name}-repository-file`) : undefined;

  return (
    <Select value={(value as string) ?? ""} onValueChange={(val) => onChange(val || undefined)}>
      <SelectTrigger className="w-full" data-testid={testId}>
        <SelectValue placeholder={isLoading ? "Loading files..." : `Select ${field.label || "file"}`} />
      </SelectTrigger>
      <SelectContent className="max-h-60">
        {files.map((filePath) => (
          <SelectItem key={filePath} value={filePath}>
            {filePath}
          </SelectItem>
        ))}
        {isError && <div className="px-2 py-1.5 text-sm text-red-500">Failed to load files</div>}
        {!isLoading && !isError && files.length === 0 && (
          <div className="px-2 py-1.5 text-sm text-slate-500">No files found</div>
        )}
      </SelectContent>
    </Select>
  );
};
