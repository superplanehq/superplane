import { Badge } from "../../../components/ui/badge";

export function NodeCountLabel({ components, triggers }: { components: number; triggers: number }) {
  const parts: string[] = [];
  if (components > 0) parts.push(`${components} ${components === 1 ? "component" : "components"}`);
  if (triggers > 0) parts.push(`${triggers} ${triggers === 1 ? "trigger" : "triggers"}`);
  if (parts.length === 0) return null;
  return <div className="text-xs text-gray-500 dark:text-gray-500 mt-2">{parts.join(" · ")}</div>;
}

export function TagBadges({ tags }: { tags: string[] }) {
  if (tags.length === 0) return <div />;
  return (
    <div className="flex flex-wrap gap-1">
      {tags.map((tag) => (
        <Badge key={tag} variant="outline" className="text-[11px] px-1.5 py-0 text-gray-600 dark:text-gray-400">
          {tag}
        </Badge>
      ))}
    </div>
  );
}
