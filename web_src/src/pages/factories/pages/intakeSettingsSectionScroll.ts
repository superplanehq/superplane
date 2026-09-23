export interface IntakeSettingsScrollMetrics {
  scrollTop: number;
  clientHeight: number;
  scrollHeight: number;
  paddingTop: number;
}

export interface IntakeSettingsSectionOffset<Id extends string = string> {
  id: Id;
  top: number;
}

const TOP_SCROLL_EPSILON = 1;
const BOTTOM_SCROLL_EPSILON = 1;

export function intakeSettingsSectionOffset(container: HTMLElement, target: HTMLElement): number {
  return target.getBoundingClientRect().top - container.getBoundingClientRect().top + container.scrollTop;
}

export function intakeSettingsActiveSectionId<Id extends string>(
  metrics: IntakeSettingsScrollMetrics,
  sections: readonly IntakeSettingsSectionOffset<Id>[],
): Id | null {
  if (sections.length === 0) {
    return null;
  }

  if (metrics.scrollHeight <= metrics.clientHeight) {
    return null;
  }

  if (metrics.scrollTop < TOP_SCROLL_EPSILON) {
    return sections[0].id;
  }

  if (metrics.scrollTop + metrics.clientHeight >= metrics.scrollHeight - BOTTOM_SCROLL_EPSILON) {
    return sections[sections.length - 1].id;
  }

  const spyLine = metrics.scrollTop + metrics.paddingTop;
  let activeId = sections[0].id;
  for (const section of sections) {
    if (section.top <= spyLine) {
      activeId = section.id;
    }
  }
  return activeId;
}

export function intakeSettingsClickScrollTop(section: { top: number; isFirst: boolean }, paddingTop: number): number {
  if (section.isFirst) {
    return 0;
  }
  return Math.max(0, section.top - paddingTop);
}
