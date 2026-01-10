import React from "react";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../select";
import { FieldRendererProps } from "./types";

type Kind = "branch" | "tag";

function parseGitRef(ref?: string): { kind: Kind; name: string } {
  const val = (ref || "").trim();
  if (val.startsWith("refs/heads/")) {
    return { kind: "branch", name: val.replace(/^refs\/heads\//, "") };
  }
  if (val.startsWith("ref/heads/")) {
    // Be tolerant of older placeholder without the trailing 's'
    return { kind: "branch", name: val.replace(/^ref\/heads\//, "") };
  }
  if (val.startsWith("refs/tags/")) {
    return { kind: "tag", name: val.replace(/^refs\/tags\//, "") };
  }
  if (val.startsWith("ref/tags/")) {
    return { kind: "tag", name: val.replace(/^ref\/tags\//, "") };
  }

  // Default to branch if unknown; keep whatever name is there
  return { kind: "branch", name: val };
}

function buildGitRef(kind: Kind, name: string): string {
  const sanitized = (name || "").trim();
  if (sanitized === "") return "";
  if (kind === "tag") return `refs/tags/${sanitized}`;
  return `refs/heads/${sanitized}`;
}

export const GitRefFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  const effective = (value as string) ?? (field.defaultValue as string) ?? "";
  const initial = React.useMemo(() => parseGitRef(effective), [effective]);

  const [kind, setKind] = React.useState<Kind>(initial.kind);
  const [name, setName] = React.useState<string>(initial.name);

  // Keep local state in sync if external value/default changes
  React.useEffect(() => {
    setKind(initial.kind);
    setName(initial.name);
  }, [initial.kind, initial.name]);

  const update = (nextKind: Kind, nextName: string) => {
    setKind(nextKind);
    setName(nextName);
    const ref = buildGitRef(nextKind, nextName);
    onChange(ref !== "" ? ref : undefined);
  };

  return (
    <div className="flex gap-2">
      <div className="w-40 min-w-32">
        <Select value={kind} onValueChange={(v) => update((v as Kind) || "branch", name)}>
          <SelectTrigger className={`w-full ${hasError ? "border-red-500 border-2" : ""}`}>
            <SelectValue placeholder="Reference type" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="branch">Branch</SelectItem>
            <SelectItem value="tag">Tag</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div className="flex-1">
        <Input
          type="text"
          value={name}
          onChange={(e) => update(kind, e.target.value)}
          placeholder={field.placeholder || (kind === "tag" ? "e.g. v1.0.0" : "e.g. main")}
          className={hasError ? "border-red-500 border-2" : ""}
        />
      </div>
    </div>
  );
};
