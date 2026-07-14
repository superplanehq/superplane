import React from "react";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { buildGitRef, gitRefPlaceholder, parseGitRef, type GitRefKind } from "@/lib/gitRef";
import type { FieldRendererProps } from "./types";

export const GitRefFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange }) => {
  const effective = (value as string) ?? (field.defaultValue as string) ?? "";
  const initial = React.useMemo(() => parseGitRef(effective), [effective]);

  const [kind, setKind] = React.useState<GitRefKind>(initial.kind);
  const [name, setName] = React.useState<string>(initial.name);

  // Keep local state in sync if external value/default changes
  React.useEffect(() => {
    setKind(initial.kind);
    setName(initial.name);
  }, [initial.kind, initial.name]);

  const update = (nextKind: GitRefKind, nextName: string) => {
    setKind(nextKind);
    setName(nextName);
    const ref = buildGitRef(nextKind, nextName);
    onChange(ref !== "" ? ref : undefined);
  };

  return (
    <div className="flex gap-2">
      <div className="w-40 min-w-32">
        <Select value={kind} onValueChange={(v) => update((v as GitRefKind) || "branch", name)}>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Reference type" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="branch">Branch</SelectItem>
            <SelectItem value="tag">Tag</SelectItem>
            <SelectItem value="pull-request">Pull request</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div className="flex-1">
        <Input
          type="text"
          value={name}
          onChange={(e) => update(kind, e.target.value)}
          placeholder={field.placeholder || gitRefPlaceholder(kind)}
          className=""
        />
      </div>
    </div>
  );
};
