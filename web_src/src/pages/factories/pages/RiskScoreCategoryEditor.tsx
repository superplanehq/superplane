import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Plus, Trash2 } from "lucide-react";
import { useEffect, useState, type CSSProperties } from "react";

import {
  isRiskScoreCategoryName,
  nextRiskScoreCategoryId,
  type RiskScoreCategory,
  type RiskScoreLevel,
} from "./riskScoreCategories";
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
    if (
      !isRiskScoreCategoryName(name) ||
      categories.some((category) => category.name.toLowerCase() === name.toLowerCase())
    ) {
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
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              className="shrink-0 text-muted-foreground"
              aria-label={`${RISK_SCORE_SETUP_COPY.remove} ${category.name}`}
              data-testid={`risk-score-category-remove-${category.id}`}
              onClick={() => onChange(categories.filter((entry) => entry.id !== category.id))}
            >
              <Trash2 className="size-3.5" aria-hidden />
            </Button>
          </li>
        ))}
      </ul>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        className="mt-3 text-muted-foreground"
        onClick={() => setAdding(true)}
        data-testid="risk-score-category-add-more"
      >
        <Plus className="size-3.5" aria-hidden />
        {RISK_SCORE_SETUP_COPY.addMore}
      </Button>
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
  const [score, setScore] = useState<RiskScoreLevel>(2);
  const trimmed = name.trim();
  const duplicate = existingNames.some((existing) => existing.toLowerCase() === trimmed.toLowerCase());
  const invalid = trimmed.length > 0 && !isRiskScoreCategoryName(trimmed);

  useEffect(() => {
    if (!open) {
      setName("");
      setScore(2);
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
          <div className="space-y-1.5">
            <Label htmlFor="risk-score-category-name" className="text-[13px]">
              {RISK_SCORE_SETUP_COPY.addNameLabel}
            </Label>
            <Input
              id="risk-score-category-name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              autoFocus
              data-testid="risk-score-category-name"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="risk-score-category-score" className="text-[13px]">
              {RISK_SCORE_SETUP_COPY.addScoreLabel}
            </Label>
            <RiskLevelSelect
              id="risk-score-category-score"
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
          {invalid ? (
            <p className="text-[13px] text-destructive" role="alert">
              {RISK_SCORE_SETUP_COPY.addInvalid}
            </p>
          ) : null}
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {RISK_SCORE_SETUP_COPY.cancel}
          </Button>
          <Button
            type="button"
            disabled={trimmed.length === 0 || duplicate || invalid}
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
  id,
  value,
  label,
  testId,
  onScore,
}: {
  id?: string;
  value: RiskScoreLevel;
  label: string;
  testId: string;
  onScore: (score: RiskScoreLevel) => void;
}) {
  const tone = RISK_SCORE_SETUP_COPY.scale.find((level) => level.score === value)?.tone ?? "low";

  return (
    <Select value={String(value)} onValueChange={(next) => onScore(Number(next) as RiskScoreLevel)}>
      <SelectTrigger
        id={id}
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
