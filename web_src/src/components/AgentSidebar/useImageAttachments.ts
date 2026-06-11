import { useCallback, useRef, useState } from "react";
import { showErrorToast } from "@/lib/toast";

export const MAX_IMAGE_ATTACHMENTS = 8;
export const ALLOWED_IMAGE_TYPES = ["image/png", "image/jpeg", "image/gif", "image/webp"];

// The gRPC server enforces a default 4 MiB receive limit on the whole request
// (see pkg/grpc/server.go). Images travel as base64 (~4/3 larger than the raw
// bytes) alongside the message text, so we cap the combined raw image bytes
// below that ceiling — leaving headroom for the text and protobuf framing — to
// avoid the request being rejected at the transport layer with an HTTP 429.
const GRPC_MAX_REQUEST_BYTES = 4 * 1024 * 1024;
const REQUEST_OVERHEAD_BYTES = 256 * 1024;
export const MAX_TOTAL_IMAGE_BYTES = Math.floor(((GRPC_MAX_REQUEST_BYTES - REQUEST_OVERHEAD_BYTES) * 3) / 4);
// A single image may use the entire per-message budget.
export const MAX_IMAGE_BYTES = MAX_TOTAL_IMAGE_BYTES;

export type ComposerImage = {
  id: string;
  name: string;
  mediaType: string;
  // Raw (decoded) byte size, used to enforce the per-message payload budget.
  bytes: number;
  // Full `data:<mediaType>;base64,<data>` URL, used for inline previews.
  dataUrl: string;
  // Base64 payload only, without the data URI prefix, sent to the API.
  data: string;
};

export type UseImageAttachmentsReturn = {
  images: ComposerImage[];
  addFiles: (files: FileList | File[]) => Promise<void>;
  removeImage: (id: string) => void;
  clear: () => void;
};

// isSupportedImageFile reports whether the composer handles a file at all. Size
// limits are enforced in addFiles (with user feedback), so callers can use this
// to decide whether to intercept a paste regardless of the file's size.
export function isSupportedImageFile(file: File): boolean {
  return ALLOWED_IMAGE_TYPES.includes(file.type) && file.size > 0;
}

export function useImageAttachments(): UseImageAttachmentsReturn {
  const [images, setImages] = useState<ComposerImage[]>([]);
  const imagesRef = useRef(images);
  imagesRef.current = images;

  const addFiles = useCallback(async (files: FileList | File[]) => {
    const candidates = Array.from(files).filter(isSupportedImageFile);
    if (candidates.length === 0) return;

    const sized = candidates.filter((file) => file.size <= MAX_IMAGE_BYTES);
    if (sized.length < candidates.length) {
      showErrorToast(`Each image must be ${formatMegabytes(MAX_IMAGE_BYTES)} or smaller.`);
    }
    if (sized.length === 0) return;

    // A single unreadable file must not drop the rest of the batch.
    const read = (await Promise.all(sized.map(readImage))).filter((image): image is ComposerImage => image !== null);
    if (read.length === 0) return;

    const current = imagesRef.current;
    const accepted: ComposerImage[] = [];
    let count = current.length;
    let total = current.reduce((sum, image) => sum + image.bytes, 0);
    let rejected = false;
    for (const image of read) {
      if (count >= MAX_IMAGE_ATTACHMENTS || total + image.bytes > MAX_TOTAL_IMAGE_BYTES) {
        rejected = true;
        continue;
      }
      accepted.push(image);
      count += 1;
      total += image.bytes;
    }
    if (rejected) {
      showErrorToast(
        `Attachments are limited to ${MAX_IMAGE_ATTACHMENTS} images and ${formatMegabytes(MAX_TOTAL_IMAGE_BYTES)} per message.`,
      );
    }
    if (accepted.length > 0) {
      setImages((previous) => [...previous, ...accepted]);
    }
  }, []);

  const removeImage = useCallback((id: string) => {
    setImages((current) => current.filter((image) => image.id !== id));
  }, []);

  const clear = useCallback(() => setImages([]), []);

  return { images, addFiles, removeImage, clear };
}

function readImage(file: File): Promise<ComposerImage | null> {
  return new Promise((resolve) => {
    const reader = new FileReader();
    reader.onerror = () => resolve(null);
    reader.onload = () => {
      const dataUrl = String(reader.result);
      resolve({
        id: crypto.randomUUID(),
        name: file.name || "image",
        mediaType: file.type,
        bytes: file.size,
        dataUrl,
        data: dataUrl.slice(dataUrl.indexOf(",") + 1),
      });
    };
    reader.readAsDataURL(file);
  });
}

function formatMegabytes(bytes: number): string {
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
