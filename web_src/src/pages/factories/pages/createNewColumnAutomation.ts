import { isCanvasNameAlreadyExistsError, uniqueCanvasName } from "@/pages/home/uniqueCanvasName";

export const NEW_COLUMN_AUTOMATION_NAME = "New Automation";

const MAX_NAME_RETRY_ATTEMPTS = 20;

type CreateCanvasResult = {
  data?: {
    canvas?: {
      metadata?: {
        id?: string;
      };
    };
  };
};

type CreateCanvasInput = {
  name: string;
  description?: string;
  factoryId?: string;
  method?: "ui" | "cli" | "yaml_import" | "template";
};

export async function createNewColumnAutomationCanvas(args: {
  factoryId: string;
  existingNames: Iterable<string>;
  createCanvas: (input: CreateCanvasInput) => Promise<CreateCanvasResult>;
}): Promise<string> {
  const existing = new Set(
    [...args.existingNames].map((name) => name.trim()).filter((name): name is string => Boolean(name)),
  );
  let name = uniqueCanvasName(NEW_COLUMN_AUTOMATION_NAME, existing);

  for (let attempt = 0; attempt < MAX_NAME_RETRY_ATTEMPTS; attempt++) {
    try {
      const result = await args.createCanvas({
        name,
        description: "",
        factoryId: args.factoryId,
        method: "ui",
      });
      const canvasId = result.data?.canvas?.metadata?.id?.trim();
      if (!canvasId) {
        throw new Error("Failed to create automation");
      }
      return canvasId;
    } catch (error) {
      if (!isCanvasNameAlreadyExistsError(error)) {
        throw error;
      }
      existing.add(name);
      name = uniqueCanvasName(NEW_COLUMN_AUTOMATION_NAME, existing);
    }
  }

  throw new Error("Failed to create automation");
}
