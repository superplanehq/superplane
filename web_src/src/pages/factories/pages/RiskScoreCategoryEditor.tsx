import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Plus, Trash2 } from "lucide-react";
import { useEffect, useState, type CSSProperties } from "react";

import { nextRiskScoreCategoryId, type RiskScoreCategory, type RiskScoreLevel } from "./riskScoreCategories";
import { RISK_SCORE_SETUP_COPY } from "./riskScoreSetupCopy";

export function RiskScoreCategoryEditor({
  categories,
  onChange,
}: {
  categories: RiskScoreCategory[];
  onChange: (next: RiskScoreCategory[]) => void;
}) {
  const [adding, setAdding] = useState(false);

  const add = (name: string, score: RiskScoreLevel) => {
    if (!name || categories.some((category) => category.name.toLowerCase() === name.toLowerCase())) {
      return;
    }
    onChange([...categories, { id: nextRiskScoreCategoryId(categories), name, score }]);
  };

  return (
    <section>
      <ul className="divide-y divide-border rounded-lg border border-border" data-testid="risk-score-setup-categories">
        {categories.map((category) => (
          <li key={category.id} className="flex items-center gap-2 px-3 py-2">
            <span className="min-w-0 flex-1 text-[13px] font-medium text-foreground">{category.name}</span>
            <RiskLevelSelect
              value={category.score}
              label={category.name}
              testId={`risk-score-category-${category.id}`}
              onScore={(score) =>
                onChange(categories.map((entry) => (entry.id === category.id ? { ...entry, score } : entry)))
              }
            />
            <button
              type="button"
              className="flex size-8 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-foreground"
              aria-label={`${RISK_SCORE_SETUP_COPY.remove} ${category.name}`}
              data-testid={`risk-score-category-remove-${category.id}`}
              onClick={() => onChange(categories.filter((entry) => entry.id !== category.id))}
            >
              <Trash2 className="size-3.5" aria-hidden />
            </button>
          </li>
        ))}
      </ul>
      <button
        type="button"
        className="mt-3 inline-flex items-center gap-1 text-[13px] font-medium text-muted-foreground hover:text-foreground"
        onClick={() => setAdding(true)}
        data-testid="risk-score-category-add-more"
      >
        <Plus className="size-3.5" aria-hidden />
        {RISK_SCORE_SETUP_COPY.addMore}
      </button>
      <AddCategoryDialog
        open={adding}
        existingNames={categories.map((category) => category.name)}
        onClose={() => setAdding(false)}
        onAdd={(name, score) => {
          add(name, score);
          setAdding(false);
        }}
      />
    </section>
  );
}

function AddCategoryDialog({
  open,
  existingNames,
  onClose,
  onAdd,
}: {
  open: boolean;
  existingNames: string[];
  onClose: () => void;
  onAdd: (name: string, score: RiskScoreLevel) => void;
}) {
  const [name, setName] = useState("");
  const [score, setScore] = useState<RiskScoreLevel>(3);
  const trimmed = name.trim();
  const duplicate = existingNames.some((existing) => existing.toLowerCase() === trimmed.toLowerCase());

  useEffect(() => {
    if (!open) {
      setName("");
      setScore(3);
    }
  }, [open]);

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          onClose();
        }
      }}
    >
      <DialogContent className="sm:max-w-md" data-testid="risk-score-category-dialog">
        <DialogHeader>
          <DialogTitle>{RISK_SCORE_SETUP_COPY.addTitle}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <label className="block space-y-1.5">
            <span className="text-[13px] font-medium">{RISK_SCORE_SETUP_COPY.addNameLabel}</span>
            <Input
              value={name}
              onChange={(event) => setName(event.target.value)}
              autoFocus
              data-testid="risk-score-category-name"
            />
          </label>
          <div className="space-y-1.5">
            <span className="text-[13px] font-medium">{RISK_SCORE_SETUP_COPY.addScoreLabel}</span>
            <RiskLevelSelect
              value={score}
              label={RISK_SCORE_SETUP_COPY.addScoreLabel}
              testId="risk-score-category-score"
              onScore={setScore}
            />
          </div>
          {duplicate ? (
            <p className="text-[13px] text-destructive" role="alert">
              {RISK_SCORE_SETUP_COPY.addDuplicate}
            </p>
          ) : null}
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {RISK_SCORE_SETUP_COPY.cancel}
          </Button>
          <Button
            type="button"
            disabled={trimmed.length === 0 || duplicate}
            onClick={() => onAdd(trimmed, score)}
            data-testid="risk-score-category-add"
          >
            {RISK_SCORE_SETUP_COPY.addButton}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function RiskLevelSelect({
  value,
  label,
  testId,
  onScore,
}: {
  value: RiskScoreLevel;
  label: string;
  testId: string;
  onScore: (score: RiskScoreLevel) => void;
}) {
  const tone = RISK_SCORE_SETUP_COPY.scale.find((level) => level.score === value)?.tone ?? "low";

  return (
    <Select value={String(value)} onValueChange={(next) => onScore(Number(next) as RiskScoreLevel)}>
      <SelectTrigger
        className="h-8 w-[9.75rem] shrink-0 font-medium"
        style={riskToneStyle(tone)}
        data-testid={testId}
        data-tone={tone}
        aria-label={label}
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {RISK_SCORE_SETUP_COPY.scale.map((level) => (
          <SelectItem key={level.score} value={String(level.score)}>
            <span style={{ color: riskToneStyle(level.tone).color }}>{level.status}</span>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

function riskToneStyle(tone: "low" | "caution" | "critical"): CSSProperties {
  if (tone === "critical") {
    return { color: "#b91c1c", borderColor: "#fca5a5", backgroundColor: "#fef2f2" };
  }
  if (tone === "caution") {
    return { color: "#b45309", borderColor: "#fcd34d", backgroundColor: "#fffbeb" };
  }
  return { color: "#047857", borderColor: "#6ee7b7", backgroundColor: "#ecfdf5" };
}
