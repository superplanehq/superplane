import { filesCreateFactoryFile, filesCreateWorkOrderFile } from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import {
  isAllowedWorkOrderFile,
  isInlineWorkOrderImage,
  MAX_WORK_ORDER_FILE_BYTES,
  resolveWorkOrderFileMimeType,
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

type WorkOrderFileUploadTarget = {
  organizationId: string;
  factoryId: string;
  orderId?: string;
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
  args: WorkOrderFileUploadTarget,
  file: File,
): Promise<UploadedWorkOrderFile | null> {
  const contentType = resolveWorkOrderFileMimeType(file);
  if (!isAllowedWorkOrderFile(file)) {
    showErrorToast("This file type is not allowed.");
    return null;
  }
  if (file.size > MAX_WORK_ORDER_FILE_BYTES) {
    showErrorToast("Each file must be 10 MB or smaller.");
    return null;
  }

  try {
    const pendingFile = await createPendingWorkOrderFile(args, file.name, contentType);
    if (!pendingFile) {
      return null;
    }
    const { id, uploadUrl } = pendingFile;

    const response = await fetch(uploadUrl, {
      method: "PUT",
      credentials: "include",
      headers: {
        "x-organization-id": args.organizationId,
        "Content-Type": contentType || "application/octet-stream",
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
      contentType,
      ref: workOrderFileRef(id),
      previewUrl,
      isImage: isInlineWorkOrderImage(contentType),
    };
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, "The file could not be stored."));
    return null;
  }
}

async function createPendingWorkOrderFile(
  args: WorkOrderFileUploadTarget,
  filename: string,
  contentType: string,
): Promise<{ id: string; uploadUrl: string } | null> {
  const created = args.orderId
    ? await filesCreateWorkOrderFile<false>(
        withOrganizationHeader({
          organizationId: args.organizationId,
          path: { factoryId: args.factoryId, orderId: args.orderId },
          body: { filename, contentType },
          throwOnError: false,
        }),
      )
    : await filesCreateFactoryFile<false>(
        withOrganizationHeader({
          organizationId: args.organizationId,
          path: { factoryId: args.factoryId },
          body: { filename, contentType },
          throwOnError: false,
        }),
      );

  if (created.error !== undefined) {
    showErrorToast(workOrderFileCreateFailureMessage(created.error, created.response?.status));
    return null;
  }

  const id = created.data?.file?.id;
  const uploadUrl = created.data?.file?.uploadUrl;
  if (!id || !uploadUrl) {
    showErrorToast("The file could not be stored.");
    return null;
  }

  return { id, uploadUrl };
}

const attachPermissionMessage = "You do not have permission to attach files here.";
const missingAttachTargetMessage = "This workspace or task no longer exists. Refresh the page and try again.";
const missingResourceApiMessage = "resource not found";

function workOrderFileCreateFailureMessage(error: unknown, status: number | undefined): string {
  if (isAttachPermissionFailure(error, status)) {
    return attachPermissionMessage;
  }
  if (status === 404) {
    return missingAttachTargetMessage;
  }
  return getApiErrorMessage(error, "The file could not be stored.");
}

function isAttachPermissionFailure(error: unknown, status: number | undefined): boolean {
  if (status === 403) {
    return true;
  }
  if (status !== 404) {
    return false;
  }
  return normalizedApiMessage(error) !== missingResourceApiMessage;
}

function normalizedApiMessage(error: unknown): string {
  return getApiErrorMessage(error, "").trim().toLowerCase();
}
