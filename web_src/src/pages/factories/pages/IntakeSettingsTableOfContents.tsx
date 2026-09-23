import { cn } from "@/lib/utils";
import { useEffect, useRef, useState, type RefObject } from "react";

import {
  intakeSettingsActiveSectionId,
  intakeSettingsClickScrollTop,
  intakeSettingsSectionOffset,
  type IntakeSettingsScrollMetrics,
  type IntakeSettingsSectionOffset,
} from "./intakeSettingsSectionScroll";
import {
  intakeSettingsSectionDomId,
  type IntakeSettingsSection,
  type IntakeSettingsSectionId,
} from "./intakeSourceSettingsModel";

const PROGRAMMATIC_SCROLL_PIN_MS = 400;

function scrollMetrics(container: HTMLElement): IntakeSettingsScrollMetrics {
  return {
    scrollTop: container.scrollTop,
    clientHeight: container.clientHeight,
    scrollHeight: container.scrollHeight,
    paddingTop: Number.parseFloat(getComputedStyle(container).paddingTop) || 0,
  };
}

function sectionOffsets(
  container: HTMLElement,
  sections: readonly IntakeSettingsSection[],
): IntakeSettingsSectionOffset<IntakeSettingsSectionId>[] {
  const offsets: IntakeSettingsSectionOffset<IntakeSettingsSectionId>[] = [];
  for (const section of sections) {
    const target = document.getElementById(intakeSettingsSectionDomId(section.id));
    if (!target) {
      continue;
    }
    offsets.push({ id: section.id, top: intakeSettingsSectionOffset(container, target) });
  }
  return offsets;
}

export function IntakeSettingsTableOfContents({
  sections,
  scrollContainerRef,
}: {
  sections: readonly IntakeSettingsSection[];
  scrollContainerRef: RefObject<HTMLElement | null>;
}) {
  const [activeId, setActiveId] = useState<IntakeSettingsSectionId | null>(sections[0]?.id ?? null);
  const scrollingToRef = useRef<IntakeSettingsSectionId | null>(null);
  const scrollTimeoutRef = useRef<number | null>(null);
  const pinAbortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    setActiveId(sections[0]?.id ?? null);
  }, [sections]);

  useEffect(() => {
    const root = scrollContainerRef.current;
    if (!root || sections.length === 0) {
      return;
    }

    function recompute() {
      if (!root || scrollingToRef.current) {
        return;
      }

      const next = intakeSettingsActiveSectionId(scrollMetrics(root), sectionOffsets(root, sections));
      if (next) {
        setActiveId(next);
      }
    }

    root.addEventListener("scroll", recompute, { passive: true });
    const resizeObserver = new ResizeObserver(recompute);
    resizeObserver.observe(root);
    for (const section of sections) {
      const target = document.getElementById(intakeSettingsSectionDomId(section.id));
      if (target) {
        resizeObserver.observe(target);
      }
    }
    recompute();

    return () => {
      root.removeEventListener("scroll", recompute);
      resizeObserver.disconnect();
    };
  }, [scrollContainerRef, sections]);

  useEffect(() => {
    return () => {
      pinAbortRef.current?.abort();
      if (scrollTimeoutRef.current !== null) {
        window.clearTimeout(scrollTimeoutRef.current);
      }
    };
  }, []);

  if (sections.length === 0) {
    return null;
  }

  function scrollTo(sectionId: IntakeSettingsSectionId) {
    const container = scrollContainerRef.current;
    const target = document.getElementById(intakeSettingsSectionDomId(sectionId));
    if (!container || !target) {
      return;
    }

    pinAbortRef.current?.abort();
    const pinAbort = new AbortController();
    pinAbortRef.current = pinAbort;

    scrollingToRef.current = sectionId;
    setActiveId(sectionId);

    const metrics = scrollMetrics(container);
    container.scrollTo({
      top: intakeSettingsClickScrollTop(
        { top: intakeSettingsSectionOffset(container, target), isFirst: sectionId === sections[0]?.id },
        metrics.paddingTop,
      ),
      behavior: "smooth",
    });

    function clearPin() {
      if (scrollTimeoutRef.current !== null) {
        window.clearTimeout(scrollTimeoutRef.current);
        scrollTimeoutRef.current = null;
      }
      pinAbort.abort();
      pinAbortRef.current = null;
      scrollingToRef.current = null;
    }

    container.addEventListener("scrollend", clearPin, { once: true, signal: pinAbort.signal });
    if (scrollTimeoutRef.current !== null) {
      window.clearTimeout(scrollTimeoutRef.current);
    }
    scrollTimeoutRef.current = window.setTimeout(clearPin, PROGRAMMATIC_SCROLL_PIN_MS);
  }

  return (
    <aside
      className="hidden w-44 shrink-0 border-r border-sidebar-border bg-sidebar py-6 pl-5 pr-3 sm:block lg:w-48"
      data-testid="intake-settings-toc"
    >
      <nav aria-label="Settings sections" className="flex flex-col gap-0.5">
        {sections.map((section) => {
          const selected = section.id === activeId;
          return (
            <button
              key={section.id}
              type="button"
              aria-current={selected ? "true" : undefined}
              data-testid={`intake-settings-toc-${section.id}`}
              className={cn(
                "rounded-md px-2.5 py-1.5 text-left text-[13px] font-medium leading-snug tracking-[-0.01em]",
                selected
                  ? "bg-sidebar-accent text-sidebar-accent-foreground"
                  : "text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
              )}
              onClick={() => scrollTo(section.id)}
            >
              {section.label}
            </button>
          );
        })}
      </nav>
    </aside>
  );
}
