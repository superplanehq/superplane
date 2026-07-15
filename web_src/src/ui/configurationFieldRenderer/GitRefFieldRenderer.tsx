import { createPortal } from "react-dom";
import React from "react";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { AutoCompleteInput } from "@/components/AutoCompleteInput/AutoCompleteInput";
import { SegmentedNav } from "@/ui/SegmentedNav";
import { buildGitRef, gitRefPlaceholder, parseGitRef, type GitRefKind } from "@/lib/gitRef";
import type { FieldRendererProps } from "./types";
import { toTestId } from "@/lib/testID";
import type { RefObject } from "react";

interface GitRefFieldRendererProps extends FieldRendererProps {
  labelRightRef?: RefObject<HTMLDivElement | null>;
  labelRightReady?: boolean;
}

/**
 * Detect if value looks like a wrapped expression (e.g. {{ $["node-name"].value }}).
 * Requires both {{ and }} so fixed refs are not misclassified.
 */
function isExpressionValue(value: string | undefined): boolean {
  if (value == null) return false;
  const trimmed = value.trim();
  if (!trimmed.length) return false;
  return /\{\{[\s\S]*?\}\}/.test(trimmed);
}

export const GitRefFieldRenderer: React.FC<GitRefFieldRendererProps> = ({
  field,
  value,
  onChange,
  allowExpressions = false,
  autocompleteExampleObj = null,
  labelRightRef,
  labelRightReady = false,
}) => {
  const effective = (value as string) ?? (field.defaultValue as string) ?? "";
  const initialIsExpression = allowExpressions && isExpressionValue(effective);
  const [useExpressionMode, setUseExpressionMode] = React.useState(initialIsExpression);

  const initial = React.useMemo(() => parseGitRef(effective), [effective]);

  const [kind, setKind] = React.useState<GitRefKind>(initial.kind);
  const [name, setName] = React.useState<string>(initial.name);

  // Keep local state in sync if external value/default changes
  React.useEffect(() => {
    if (allowExpressions && isExpressionValue(effective)) {
      setUseExpressionMode(true);
      return;
    }

    setKind(initial.kind);
    setName(initial.name);
  }, [allowExpressions, effective, initial.kind, initial.name]);

  const update = (nextKind: GitRefKind, nextName: string) => {
    setKind(nextKind);
    setName(nextName);
    const ref = buildGitRef(nextKind, nextName);
    onChange(ref !== "" ? ref : undefined);
  };

  const fixedPicker = (
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

  const expressionInput = (
    <AutoCompleteInput
      exampleObj={autocompleteExampleObj}
      value={effective}
      onChange={(nextValue) => onChange(nextValue || undefined)}
      placeholder={field.placeholder ?? 'e.g. {{ $["node-name"].data.ref }}'}
      startWord="{{"
      prefix="{{ "
      suffix=" }}"
      inputSize="md"
      showValuePreview
      quickTip="Tip: type `{{` to start an expression."
      className=""
    />
  );

  if (!allowExpressions) {
    return <div data-testid={toTestId(`git-ref-field-${field.name}`)}>{fixedPicker}</div>;
  }

  const modeToggle = (
    <SegmentedNav
      ariaLabel="Value mode"
      value={useExpressionMode ? "expression" : "fixed"}
      onValueChange={(nextValue) => {
        setUseExpressionMode(nextValue === "expression");
      }}
      options={[
        { value: "fixed", label: "Fixed" },
        { value: "expression", label: "Expression" },
      ]}
      size="xs"
    />
  );
  const toggleInLabelRow =
    labelRightReady && labelRightRef?.current ? createPortal(modeToggle, labelRightRef.current) : null;

  return (
    <div data-testid={toTestId(`git-ref-field-${field.name}`)} className="space-y-2">
      {toggleInLabelRow ?? <div className="flex justify-end">{modeToggle}</div>}
      {useExpressionMode ? expressionInput : fixedPicker}
    </div>
  );
};
