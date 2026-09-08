import { filesCreateFactoryFile, filesCreateWorkOrderFile } from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import {
  isAllowedWorkOrderFile,
  isInlineWorkOrderImage,
  MAX_WORK_ORDER_FILE_BYTES,
  setWorkOrderFilePreviewUrl,
  workOrderFileRef,
} from "@/lib/workOrderFiles";
import { useCallback, useState } from "react";

export type UploadedWorkOrderFile = {
  id: string;
  filename: string;
  contentType: string;
  ref: string;
  previewUrl: string;
  isImage: boolean;
};

export function useWorkOrderFileUpload({
  organizationId,
  factoryId,
  orderId,
}: {
  organizationId: string;
  factoryId: string;
  orderId?: string;
}) {
  const [isUploading, setIsUploading] = useState(false);

  const uploadFiles = useCallback(
    async (files: FileList | File[]): Promise<UploadedWorkOrderFile[]> => {
      const candidates = Array.from(files);
      if (candidates.length === 0) {
        return [];
      }

      const uploaded: UploadedWorkOrderFile[] = [];
      setIsUploading(true);
      try {
        for (const file of candidates) {
          const next = await uploadOneWorkOrderFile({ organizationId, factoryId, orderId }, file);
          if (next) {
            uploaded.push(next);
          }
        }
      } finally {
        setIsUploading(false);
      }
      return uploaded;
    },
    [factoryId, orderId, organizationId],
  );

  return { uploadFiles, isUploading };
}

async function uploadOneWorkOrderFile(
  args: { organizationId: string; factoryId: string; orderId?: string },
  file: File,
): Promise<UploadedWorkOrderFile | null> {
  if (!isAllowedWorkOrderFile(file)) {
    showErrorToast("This file type is not allowed.");
    return null;
  }
  if (file.size > MAX_WORK_ORDER_FILE_BYTES) {
    showErrorToast("Each file must be 10 MB or smaller.");
    return null;
  }

  try {
    const created = args.orderId
      ? await filesCreateWorkOrderFile(
          withOrganizationHeader({
            organizationId: args.organizationId,
            path: { factoryId: args.factoryId, orderId: args.orderId },
            body: { filename: file.name, contentType: file.type },
          }),
        )
      : await filesCreateFactoryFile(
          withOrganizationHeader({
            organizationId: args.organizationId,
            path: { factoryId: args.factoryId },
            body: { filename: file.name, contentType: file.type },
          }),
        );
    const id = created.data?.file?.id;
    if (!id) {
      showErrorToast("The file could not be stored.");
      return null;
    }

    const response = await fetch(`/api/v1/files/${id}/content`, {
      method: "PUT",
      credentials: "include",
      headers: {
        "x-organization-id": args.organizationId,
        "Content-Type": file.type || "application/octet-stream",
      },
      body: file,
    });
    if (!response.ok) {
      showErrorToast("The file could not be stored.");
      return null;
    }

    const previewUrl = URL.createObjectURL(file);
    setWorkOrderFilePreviewUrl(id, previewUrl);
    return {
      id,
      filename: file.name,
      contentType: file.type,
      ref: workOrderFileRef(id),
      previewUrl,
      isImage: isInlineWorkOrderImage(file.type),
    };
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, "The file could not be stored."));
    return null;
  }
}
