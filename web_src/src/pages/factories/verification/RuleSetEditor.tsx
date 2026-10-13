import { useState } from "react";
import { Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { factoryCardClassName } from "@/pages/factories/pages/factoryPageLayoutStyles";

import type { FindingSeverity, QualityDomain, Rule, RuleEnforcement, RuleSet } from "./types";
import { QUALITY_DOMAIN_LABELS } from "./types";

const SEVERITY_OPTIONS: FindingSeverity[] = ["high", "medium", "low"];
const ENFORCEMENT_OPTIONS: RuleEnforcement[] = ["blocking", "advisory"];
const DOMAIN_ORDER = Object.keys(QUALITY_DOMAIN_LABELS) as QualityDomain[];

interface RuleSetEditorProps {
  value: RuleSet;
  onChange: (ruleSet: RuleSet) => void;
}

/**
 * Editor for an org-scoped rule set: rules grouped by domain with severity
 * and enforcement controls, plus a YAML preview of the same data.
 */
export function RuleSetEditor({ value, onChange }: RuleSetEditorProps) {
  const [ruleSet, setRuleSet] = useState(value);

  const update = (next: RuleSet) => {
    setRuleSet(next);
    onChange(next);
  };

  const updateRule = (ruleId: string, changes: Partial<Rule>) => {
    update({
      ...ruleSet,
      rules: ruleSet.rules.map((rule) => (rule.id === ruleId ? { ...rule, ...changes } : rule)),
    });
  };

  const addRule = () => {
    const nextIndex = ruleSet.rules.length + 1;
    update({
      ...ruleSet,
      rules: [
        ...ruleSet.rules,
        {
          id: `type-safety/new-rule-${nextIndex}`,
          name: `New rule ${nextIndex}`,
          domain: "type-safety",
          description: "Describe what this rule requires.",
          severity: "medium",
          enforcement: "advisory",
        },
      ],
    });
  };

  return (
    <section className={cn(factoryCardClassName, "flex flex-col")} aria-label="Rule set editor">
      <header className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-4 py-3">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h3 className="workspace-section-title text-foreground">Rule set</h3>
          <p className="text-[12px] text-muted-foreground">
            Rules apply to every verification suite that uses this rule set.
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={addRule}>
          <Plus className="size-4" aria-hidden />
          Add rule
        </Button>
      </header>

      <div className="grid gap-4 px-4 py-4 xl:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
        <div className="flex flex-col gap-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="rule-set-name">Name</Label>
              <Input
                id="rule-set-name"
                value={ruleSet.name}
                onChange={(event) => update({ ...ruleSet, name: event.target.value })}
                placeholder="Production"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="rule-set-description">Description</Label>
              <Input
                id="rule-set-description"
                value={ruleSet.description}
                onChange={(event) => update({ ...ruleSet, description: event.target.value })}
                placeholder="Default rules for production repositories."
              />
            </div>
          </div>

          {ruleSet.rules.length === 0 ? (
            <EmptyRulesState onAddRule={addRule} />
          ) : (
            <RulesByDomain rules={ruleSet.rules} onRuleChange={updateRule} />
          )}
        </div>

        <YamlPreview ruleSet={ruleSet} />
      </div>
    </section>
  );
}

function EmptyRulesState({ onAddRule }: { onAddRule: () => void }) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-md border border-dashed border-border bg-background px-4 py-10 text-center">
      <p className="workspace-body-text text-foreground">This rule set has no rules.</p>
      <p className="text-[13px] text-muted-foreground">Add the first rule to make verification steps enforce it.</p>
      <Button onClick={onAddRule}>
        <Plus className="size-4" aria-hidden />
        Add rule
      </Button>
    </div>
  );
}

function RulesByDomain({
  rules,
  onRuleChange,
}: {
  rules: Rule[];
  onRuleChange: (ruleId: string, changes: Partial<Rule>) => void;
}) {
  return (
    <div className="flex flex-col gap-3">
      {DOMAIN_ORDER.map((domain) => {
        const group = rules.filter((rule) => rule.domain === domain);
        if (group.length === 0) return null;
        return (
          <div key={domain} className="rounded-md border border-border bg-background">
            <p className="workspace-section-label border-b border-border px-3 py-2 text-muted-foreground">
              {QUALITY_DOMAIN_LABELS[domain]}
            </p>
            <ul className="divide-y divide-border">
              {group.map((rule) => (
                <RuleRow key={rule.id} rule={rule} onChange={onRuleChange} />
              ))}
            </ul>
          </div>
        );
      })}
    </div>
  );
}

function RuleRow({ rule, onChange }: { rule: Rule; onChange: (ruleId: string, changes: Partial<Rule>) => void }) {
  return (
    <li className="flex flex-col gap-2 px-3 py-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="text-[13px] font-medium text-foreground">{rule.name}</span>
          <code className="font-mono text-[11px] text-muted-foreground">{rule.id}</code>
        </div>
        <div className="flex items-center gap-2">
          <Select
            value={rule.severity}
            onValueChange={(severity) => onChange(rule.id, { severity: severity as FindingSeverity })}
          >
            <SelectTrigger size="sm" aria-label={`Severity for ${rule.name}`}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {SEVERITY_OPTIONS.map((severity) => (
                <SelectItem key={severity} value={severity}>
                  {severity === "high" ? "High" : severity === "medium" ? "Medium" : "Low"}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select
            value={rule.enforcement}
            onValueChange={(enforcement) => onChange(rule.id, { enforcement: enforcement as RuleEnforcement })}
          >
            <SelectTrigger size="sm" aria-label={`Enforcement for ${rule.name}`}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {ENFORCEMENT_OPTIONS.map((enforcement) => (
                <SelectItem key={enforcement} value={enforcement}>
                  {enforcement === "blocking" ? "Blocking" : "Advisory"}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
      <p className="text-[13px] text-muted-foreground">{rule.description}</p>
    </li>
  );
}

function YamlPreview({ ruleSet }: { ruleSet: RuleSet }) {
  return (
    <div className="flex min-w-0 flex-col rounded-md border border-border bg-slate-50 dark:bg-gray-900">
      <p className="workspace-section-label border-b border-border px-3 py-2 text-muted-foreground">YAML preview</p>
      <pre className="overflow-x-auto px-3 py-3 font-mono text-[12px] leading-5 text-slate-700 dark:text-gray-300">
        {ruleSetToYaml(ruleSet)}
      </pre>
    </div>
  );
}

function ruleSetToYaml(ruleSet: RuleSet): string {
  const lines = [`name: ${ruleSet.name || "(unnamed)"}`];
  if (ruleSet.description) lines.push(`description: ${ruleSet.description}`);
  lines.push("rules:");
  if (ruleSet.rules.length === 0) lines.push("  []");
  for (const rule of ruleSet.rules) {
    lines.push(`  - id: ${rule.id}`);
    lines.push(`    domain: ${rule.domain}`);
    lines.push(`    severity: ${rule.severity}`);
    lines.push(`    enforcement: ${rule.enforcement}`);
  }
  return lines.join("\n");
}
