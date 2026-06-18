import { useCallback, useRef, useState } from "react";
import { showErrorToast } from "@/lib/toast";

export const MAX_IMAGE_ATTACHMENTS = 8;
export const ALLOWED_IMAGE_TYPES = ["image/png", "image/jpeg", "image/gif", "image/webp"];
export const MAX_TOTAL_IMAGE_BYTES = 2_500_000;
export const MAX_IMAGE_BYTES = MAX_TOTAL_IMAGE_BYTES;

export type ComposerImage = {
  id: string;
  name: string;
  mediaType: string;
  bytes: number;
  dataUrl: string;
  data: string;
};

export type UseImageAttachmentsReturn = {
  images: ComposerImage[];
  addFiles: (files: FileList | File[]) => Promise<void>;
  removeImage: (id: string) => void;
  clear: () => void;
};

export function isSupportedImageFile(file: File): boolean {
  return ALLOWED_IMAGE_TYPES.includes(file.type) && file.size > 0;
}

export function useImageAttachments(): UseImageAttachmentsReturn {
  const [images, setImages] = useState<ComposerImage[]>([]);
  const imagesRef = useRef(images);

  const replaceImages = useCallback((nextImages: ComposerImage[]) => {
    imagesRef.current = nextImages;
    setImages(nextImages);
  }, []);

  const addFiles = useCallback(
    async (files: FileList | File[]) => {
      const candidates = Array.from(files).filter(isSupportedImageFile);
      if (candidates.length === 0) return;

      const sized = candidates.filter((file) => file.size <= MAX_IMAGE_BYTES);
      if (sized.length < candidates.length) {
        showErrorToast(`Each image must be ${formatMegabytes(MAX_IMAGE_BYTES)} or smaller.`);
      }
      if (sized.length === 0) return;

      const read = (await Promise.all(sized.map(readImage))).filter((image): image is ComposerImage => image !== null);
      if (read.length === 0) return;

      const { accepted, rejected } = selectAcceptedImages(imagesRef.current, read);
      if (rejected) {
        showErrorToast(
          `Attachments are limited to ${MAX_IMAGE_ATTACHMENTS} images and ${formatMegabytes(MAX_TOTAL_IMAGE_BYTES)} per message.`,
        );
      }
      if (accepted.length > 0) {
        replaceImages([...imagesRef.current, ...accepted]);
      }
    },
    [replaceImages],
  );

  const removeImage = useCallback(
    (id: string) => {
      replaceImages(imagesRef.current.filter((image) => image.id !== id));
    },
    [replaceImages],
  );

  const clear = useCallback(() => replaceImages([]), [replaceImages]);

  return { images, addFiles, removeImage, clear };
}

function selectAcceptedImages(current: ComposerImage[], candidates: ComposerImage[]) {
  const accepted: ComposerImage[] = [];
  let count = current.length;
  let total = current.reduce((sum, image) => sum + image.bytes, 0);
  let rejected = false;

  for (const image of candidates) {
    if (count >= MAX_IMAGE_ATTACHMENTS || total + image.bytes > MAX_TOTAL_IMAGE_BYTES) {
      rejected = true;
      continue;
    }
    accepted.push(image);
    count += 1;
    total += image.bytes;
  }

  return { accepted, rejected };
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
