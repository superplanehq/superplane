import { useCallback, useState } from "react";

export const MAX_IMAGE_ATTACHMENTS = 8;
export const MAX_IMAGE_BYTES = 10 * 1024 * 1024;
export const ALLOWED_IMAGE_TYPES = ["image/png", "image/jpeg", "image/gif", "image/webp"];

export type ComposerImage = {
  id: string;
  name: string;
  mediaType: string;
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

// isSupportedImageFile is the single source of truth for which files the
// composer will attach. Callers gate on it before deciding to intercept a
// paste, so unsupported or oversized files fall through to the default paste.
export function isSupportedImageFile(file: File): boolean {
  return ALLOWED_IMAGE_TYPES.includes(file.type) && file.size > 0 && file.size <= MAX_IMAGE_BYTES;
}

export function useImageAttachments(): UseImageAttachmentsReturn {
  const [images, setImages] = useState<ComposerImage[]>([]);

  const addFiles = useCallback(async (files: FileList | File[]) => {
    const candidates = Array.from(files).filter(isSupportedImageFile);
    if (candidates.length === 0) return;

    // A single unreadable file must not drop the rest of the batch.
    const read = (await Promise.all(candidates.map(readImage))).filter(
      (image): image is ComposerImage => image !== null,
    );
    if (read.length === 0) return;
    setImages((current) => [...current, ...read].slice(0, MAX_IMAGE_ATTACHMENTS));
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
        dataUrl,
        data: dataUrl.slice(dataUrl.indexOf(",") + 1),
      });
    };
    reader.readAsDataURL(file);
  });
}
