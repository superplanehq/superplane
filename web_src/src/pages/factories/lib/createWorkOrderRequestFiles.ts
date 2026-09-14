import type { UploadedWorkOrderFile } from "@/hooks/useWorkOrderFileUpload";

export function appendUploadedWorkOrderFiles(description: string, files: UploadedWorkOrderFile[]): string {
  const blocks = files.map((file) =>
    file.isImage ? `![${file.filename}](${file.ref})` : `[${file.filename}](${file.ref})`,
  );
  return [description.trimEnd(), ...blocks].filter((part) => part.length > 0).join("\n\n");
}
