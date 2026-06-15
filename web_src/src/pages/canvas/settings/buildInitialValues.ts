import type { CanvasesCanvas } from "@/api-client";
import type { SettingsValues } from "./types";

export function buildSettingsInitialValues(canvas: CanvasesCanvas | undefined): SettingsValues {
  return {
    name: canvas?.metadata?.name || "",
    description: canvas?.metadata?.description || "",
  };
}
