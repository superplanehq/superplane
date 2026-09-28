export interface DuplicateWorkOrderSource {
  title?: string | null;
  description?: string | null;
}

export interface DuplicateWorkOrderCreateInput {
  title: string;
  description: string;
}

export function duplicateWorkOrderCreateInput(source: DuplicateWorkOrderSource): DuplicateWorkOrderCreateInput {
  return {
    title: source.title ?? "",
    description: source.description ?? "",
  };
}
