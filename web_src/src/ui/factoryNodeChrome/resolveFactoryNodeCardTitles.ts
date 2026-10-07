/** Factory cards: component Label() as title, user node name as gray subtitle. */
export function resolveFactoryNodeCardTitles({
  componentLabel,
  nodeName,
  fallbackTitle,
}: {
  componentLabel?: string;
  nodeName?: string;
  fallbackTitle: string;
}): { title: string; subtitle: string | null } {
  const trimmedLabel = componentLabel?.trim();
  const title = trimmedLabel || fallbackTitle.trim() || "Component";
  const trimmedName = nodeName?.trim();
  return {
    title,
    subtitle: trimmedName || null,
  };
}
