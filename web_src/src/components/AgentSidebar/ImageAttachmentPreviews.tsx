import { X } from "lucide-react";
import type { ComposerImage } from "./useImageAttachments";

interface ImageAttachmentPreviewsProps {
  images: ComposerImage[];
  onRemove: (id: string) => void;
}

export function ImageAttachmentPreviews({ images, onRemove }: ImageAttachmentPreviewsProps) {
  if (images.length === 0) return null;

  return (
    <div className="flex flex-wrap gap-2 px-3 pt-2" data-testid="agent-image-previews">
      {images.map((image) => (
        <div
          key={image.id}
          className="group relative size-14 overflow-hidden rounded-md border border-slate-200 bg-slate-50 dark:border-gray-700 dark:bg-gray-900"
        >
          <img src={image.dataUrl} alt={image.name} className="size-full object-cover" />
          <button
            type="button"
            onClick={() => onRemove(image.id)}
            aria-label={`Remove ${image.name}`}
            className="absolute right-0.5 top-0.5 flex size-4 items-center justify-center rounded-full bg-slate-900/70 text-white opacity-0 transition-opacity group-hover:opacity-100"
          >
            <X className="size-3" aria-hidden />
          </button>
        </div>
      ))}
    </div>
  );
}
