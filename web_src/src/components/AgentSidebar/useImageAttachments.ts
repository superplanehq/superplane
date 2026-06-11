import { useCallback, useRef, useState } from "react";
import { showErrorToast } from "@/lib/toast";

export const MAX_IMAGE_ATTACHMENTS = 8;
export const ALLOWED_IMAGE_TYPES = ["image/png", "image/jpeg", "image/gif", "image/webp"];

// Caps the combined raw image bytes per message. Images are sent as base64
// (~4/3 larger) alongside the message text, so this stays well under the gRPC
// server's 4 MiB receive limit and mirrors maxChatImagePayloadBytes in
// pkg/grpc/actions/agents/send_agent_chat_message.go. Keeping it at or below the
// backend cap means oversized attachments are rejected client-side with a clear
// error instead of failing the request with an HTTP 429.
export const MAX_TOTAL_IMAGE_BYTES = 2_500_000;
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
