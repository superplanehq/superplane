import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  BlobScopeType,
  canvasesDeleteBlob,
  canvasesDescribeBlob,
  canvasesListBlobs,
  canvasesStoreBlob,
} from "@/api-client";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/ui/dialog";

type BlobScopePanelProps = {
  organizationId: string;
  scopeType: BlobScopeType;
  canvasId?: string;
  nodeId?: string;
  executionId?: string;
  enabled?: boolean;
  compact?: boolean;
};

function encodeBase64(bytes: Uint8Array): string {
  let binary = "";
  for (let idx = 0; idx < bytes.length; idx += 1) {
    binary += String.fromCharCode(bytes[idx]);
  }
  return btoa(binary);
}

function decodeBase64(base64: string): Uint8Array {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let idx = 0; idx < binary.length; idx += 1) {
    bytes[idx] = binary.charCodeAt(idx);
  }
  return bytes;
}

function downloadBlob(filename: string, bytes: Uint8Array, contentType: string | undefined): void {
  const blob = new Blob([bytes], { type: contentType || "application/octet-stream" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

export function BlobScopePanel({
  organizationId,
  scopeType,
  canvasId,
  nodeId,
  executionId,
  enabled = true,
  compact = false,
}: BlobScopePanelProps) {
  const [path, setPath] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [blobToDelete, setBlobToDelete] = useState<{ id: string; path?: string } | null>(null);
  const queryClient = useQueryClient();
  const queryKey = useMemo(
    () => ["blobs", organizationId, scopeType, canvasId || "", nodeId || "", executionId || ""],
    [organizationId, scopeType, canvasId, nodeId, executionId],
  );

  const blobsQuery = useQuery({
    queryKey,
    queryFn: async () => {
      const response = await canvasesListBlobs(
        withOrganizationHeader({
          organizationId,
          query: {
            scopeType,
            canvasId,
            nodeId,
            executionId,
          },
        }),
      );
      const blobs = response.data?.blobs || [];
      return blobs;
    },
    enabled: enabled && !!organizationId,
    retry: 5,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 8000),
    refetchOnReconnect: "always",
  });

  const storeBlobMutation = useMutation({
    mutationFn: async () => {
      if (!file) {
        throw new Error("Please select a file");
      }

      const fileBytes = new Uint8Array(await file.arrayBuffer());
      const targetPath = path.trim() || file.name;

      const response = await canvasesStoreBlob(
        withOrganizationHeader({
          organizationId,
          body: {
            scopeType,
            canvasId,
            nodeId,
            executionId,
            path: targetPath,
            content: encodeBase64(fileBytes),
            contentType: file.type || "application/octet-stream",
          },
        }),
      );
      void response;
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey });
      setPath("");
      setFile(null);
    },
  });

  const deleteBlobMutation = useMutation({
    mutationFn: async (blobId: string) => {
      await canvasesDeleteBlob(
        withOrganizationHeader({
          organizationId,
          path: { id: blobId },
        }),
      );
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey });
    },
  });

  const restoreBlobMutation = useMutation({
    mutationFn: async ({ blobId, filename }: { blobId: string; filename: string }) => {
      const response = await canvasesDescribeBlob(
        withOrganizationHeader({
          organizationId,
          path: { id: blobId },
        }),
      );
      const encodedContent = response.data?.content;
      if (!encodedContent) {
        throw new Error("Blob has no content");
      }

      const bytes = decodeBase64(encodedContent);
      downloadBlob(filename, bytes, response.data?.blob?.contentType);
    },
  });

  const handleConfirmDelete = () => {
    if (!blobToDelete) return;
    deleteBlobMutation.mutate(blobToDelete.id, {
      onSettled: () => setBlobToDelete(null),
    });
  };

  return (
    <div className={compact ? "p-3 space-y-3" : "p-6 space-y-4"}>
      <div className="rounded-md border border-gray-200 bg-white p-3 space-y-2">
        <div className="text-xs font-medium text-gray-600">Store a blob</div>
        <p className="text-xs text-gray-500">
          Path is the full blob key (including filename). Leave it empty to use the selected file name.
        </p>
        <Input
          value={path}
          onChange={(event) => setPath(event.target.value)}
          placeholder="Blob path (full key)"
        />
        <Input type="file" onChange={(event) => setFile(event.target.files?.[0] || null)} className="cursor-pointer" />
        <Button
          onClick={() => storeBlobMutation.mutate()}
          disabled={storeBlobMutation.isPending || !file}
          size="sm"
          className="w-full sm:w-auto"
        >
          {storeBlobMutation.isPending ? "Storing..." : "Store"}
        </Button>
        {storeBlobMutation.isError ? (
          <p className="text-xs text-red-600">{(storeBlobMutation.error as Error).message}</p>
        ) : null}
      </div>

      <div className="rounded-md border border-gray-200 bg-white">
        <div className="px-3 py-2 border-b border-gray-200 text-xs font-medium text-gray-600">Blobs</div>
        {blobsQuery.isLoading ? <div className="px-3 py-4 text-sm text-gray-500">Loading blobs...</div> : null}
        {blobsQuery.isError ? (
          <div className="px-3 py-4 text-sm text-red-600 flex items-center justify-between gap-3">
            <span>Failed to load blobs. The app might still be starting.</span>
            <Button size="sm" variant="outline" onClick={() => blobsQuery.refetch()} disabled={blobsQuery.isFetching}>
              Retry
            </Button>
          </div>
        ) : null}
        {!blobsQuery.isLoading && !blobsQuery.isError && (blobsQuery.data?.length || 0) === 0 ? (
          <div className="px-3 py-4 text-sm text-gray-500">No blobs found.</div>
        ) : null}
        {!blobsQuery.isLoading && !blobsQuery.isError && (blobsQuery.data?.length || 0) > 0 ? (
          <div className="divide-y divide-gray-100">
            {(blobsQuery.data || []).map((blob) => (
              <div key={blob.id} className="px-3 py-2 flex items-center justify-between gap-3">
                <div className="min-w-0">
                  <p className="text-sm font-medium text-gray-800 truncate">{blob.path || "(no path)"}</p>
                  <p className="text-xs text-gray-500 truncate">
                    {(blob.sizeBytes || "0") + " bytes"} {blob.contentType ? `- ${blob.contentType}` : ""}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => restoreBlobMutation.mutate({ blobId: blob.id || "", filename: blob.path || "blob" })}
                    disabled={!blob.id || restoreBlobMutation.isPending}
                  >
                    Restore
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => setBlobToDelete({ id: blob.id || "", path: blob.path || undefined })}
                    disabled={!blob.id || deleteBlobMutation.isPending}
                  >
                    Delete
                  </Button>
                </div>
              </div>
            ))}
          </div>
        ) : null}
      </div>
      <Dialog open={!!blobToDelete} onOpenChange={(open) => { if (!open) setBlobToDelete(null); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete "{blobToDelete?.path || "this blob"}"?</DialogTitle>
            <DialogDescription>This cannot be undone. Are you sure you want to continue?</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="destructive"
              onClick={handleConfirmDelete}
              disabled={deleteBlobMutation.isPending}
            >
              {deleteBlobMutation.isPending ? "Deleting..." : "Delete"}
            </Button>
            <Button variant="outline" onClick={() => setBlobToDelete(null)} disabled={deleteBlobMutation.isPending}>
              Cancel
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
