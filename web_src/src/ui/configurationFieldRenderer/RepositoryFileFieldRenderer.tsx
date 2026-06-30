import React from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { canvasesListCanvasRepositoryFiles } from "@/api-client/sdk.gen";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { toTestId } from "@/lib/testID";
import type { FieldRendererProps } from "./types";

// canvas.yaml and console.yaml are virtual canvas spec files: the repository
// listing always injects them, but they are materialized from the canvas
// version in the database and never committed to git. Components read attached
// files through the git-backed repository file context, so these spec files are
// not actually attachable and must be hidden from the picker.
const CANVAS_SPEC_FILES = new Set(["canvas.yaml", "console.yaml"]);

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
      return (
        response.data?.files?.map((f) => f.path ?? "").filter((path) => path && !CANVAS_SPEC_FILES.has(path)) ?? []
      );
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
