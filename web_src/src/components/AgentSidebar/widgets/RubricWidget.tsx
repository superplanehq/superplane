import { useState } from "react";
import { Button } from "@/components/ui/button";
import { ClipboardList, ChevronDown, ChevronUp, X } from "lucide-react";
import type { RubricCategory } from "./parser";

export interface RubricCriterion {
  text: string;
}

interface RubricWidgetProps {
  title: string;
  criteria: RubricCriterion[];
  categories?: RubricCategory[];
  onAction?: (text: string) => void;
  onStartBuilding?: (rubric: { title: string; criteria: string[]; categories?: RubricCategory[] }) => void;
}

export function RubricWidget({ title, criteria, categories, onAction, onStartBuilding }: RubricWidgetProps) {
  const [modalOpen, setModalOpen] = useState(false);
  const [expanded, setExpanded] = useState(false);

  const totalCriteria = criteria.length;
  const hasCategories = categories && categories.length > 0;

  // For preview: show first category or first 3 flat criteria
  const PREVIEW_COUNT = 3;
  const previewCriteria = hasCategories
    ? categories[0].criteria.slice(0, PREVIEW_COUNT)
    : criteria.slice(0, PREVIEW_COUNT);
  const hiddenCount = totalCriteria - previewCriteria.length;

  function handleStartBuilding() {
    if (onStartBuilding) {
      onStartBuilding({ title, criteria: criteria.map((c) => c.text), categories });
    } else {
      onAction?.("Start building based on this plan");
    }
  }

  return (
    <>
      <div className="my-4 rounded-lg border border-violet-200 bg-white shadow-sm overflow-hidden">
        {/* Header */}
        <div className="px-3 py-2 bg-violet-50 border-b border-violet-200 flex items-center gap-2">
          <ClipboardList size={14} className="text-violet-600 shrink-0" />
          <p className="text-xs font-semibold text-violet-900 flex-1">{title || "Build Plan"}</p>
          <span className="text-[10px] text-violet-500 font-medium">
            {hasCategories ? `${categories.length} sections · ` : ""}
            {totalCriteria} criteria
          </span>
        </div>

        {/* Preview */}
        <div className="px-3 py-2">
          {hasCategories && !expanded && (
            <p className="text-[10px] font-semibold text-violet-600 uppercase tracking-wide mb-1">
              {categories[0].heading}
            </p>
          )}

          {!expanded &&
            previewCriteria.map((c, i) => (
              <div key={i} className="flex items-start gap-2 py-0.5">
                <span className="text-violet-400 text-xs mt-0.5 shrink-0">✦</span>
                <span className="text-xs text-slate-700">{c.text}</span>
              </div>
            ))}

          {expanded && hasCategories && <CategorizedList categories={categories} />}

          {expanded &&
            !hasCategories &&
            criteria.map((c, i) => (
              <div key={i} className="flex items-start gap-2 py-0.5">
                <span className="text-violet-400 text-xs mt-0.5 shrink-0">✦</span>
                <span className="text-xs text-slate-700">{c.text}</span>
              </div>
            ))}

          {hiddenCount > 0 && !expanded && (
            <button
              type="button"
              onClick={() => setExpanded(true)}
              className="flex items-center gap-1 text-[10px] text-violet-500 hover:text-violet-700 mt-1"
            >
              <ChevronDown size={10} />+{hiddenCount} more
            </button>
          )}
          {expanded && hiddenCount > 0 && (
            <button
              type="button"
              onClick={() => setExpanded(false)}
              className="flex items-center gap-1 text-[10px] text-violet-500 hover:text-violet-700 mt-1"
            >
              <ChevronUp size={10} />
              Show less
            </button>
          )}
        </div>

        {/* Actions */}
        <div className="px-3 pb-3 pt-1 flex items-center gap-2 border-t border-violet-100">
          <Button variant="ghost" size="sm" className="text-xs text-slate-500 h-7" onClick={() => setModalOpen(true)}>
            View Full Plan
          </Button>
          <Button
            size="sm"
            className="text-xs h-7 bg-violet-600 hover:bg-violet-700 text-white ml-auto"
            onClick={handleStartBuilding}
          >
            Start Building →
          </Button>
        </div>
      </div>

      {/* Modal */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-lg max-h-[80vh] flex flex-col mx-4">
            <div className="flex items-center justify-between px-4 py-3 border-b border-slate-200">
              <div className="flex items-center gap-2">
                <ClipboardList size={16} className="text-violet-600" />
                <h2 className="text-sm font-semibold text-slate-900">{title || "Build Plan"}</h2>
              </div>
              <button type="button" onClick={() => setModalOpen(false)} className="text-slate-400 hover:text-slate-600">
                <X size={16} />
              </button>
            </div>
            <div className="overflow-y-auto p-4 flex-1">
              {hasCategories ? (
                <CategorizedList categories={categories} showNumbers />
              ) : (
                criteria.map((c, i) => (
                  <div key={i} className="flex items-start gap-2 py-1.5 border-b border-slate-50 last:border-0">
                    <span className="text-violet-500 text-sm mt-0.5 shrink-0 font-medium">{i + 1}.</span>
                    <span className="text-sm text-slate-700">{c.text}</span>
                  </div>
                ))
              )}
            </div>
            <div className="px-4 py-3 border-t border-slate-200 flex justify-end">
              <Button variant="ghost" size="sm" className="text-xs" onClick={() => setModalOpen(false)}>
                Close
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

function CategorizedList({ categories, showNumbers }: { categories: RubricCategory[]; showNumbers?: boolean }) {
  let globalIndex = 0;
  return (
    <div className="space-y-3">
      {categories.map((cat, ci) => (
        <div key={ci}>
          <p className="text-[10px] font-semibold text-violet-600 uppercase tracking-wide mb-1">{cat.heading}</p>
          {cat.criteria.map((c, i) => {
            globalIndex++;
            return (
              <div
                key={i}
                className={`flex items-start gap-2 ${showNumbers ? "py-1.5 border-b border-slate-50 last:border-0" : "py-0.5"}`}
              >
                {showNumbers ? (
                  <span className="text-violet-500 text-sm mt-0.5 shrink-0 font-medium">{globalIndex}.</span>
                ) : (
                  <span className="text-violet-400 text-xs mt-0.5 shrink-0">✦</span>
                )}
                <span className={`${showNumbers ? "text-sm" : "text-xs"} text-slate-700`}>{c.text}</span>
              </div>
            );
          })}
        </div>
      ))}
    </div>
  );
}
