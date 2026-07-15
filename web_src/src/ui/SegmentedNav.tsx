import { segmentedNavClassName, segmentedNavTabClassName, type SegmentedNavSize } from "@/lib/segmentedNav";

export type SegmentedNavOption = {
  value: string;
  label: string;
};

export function SegmentedNav({
  ariaLabel,
  value,
  onValueChange,
  options,
  size = "default",
}: {
  ariaLabel: string;
  value: string;
  onValueChange: (value: string) => void;
  options: SegmentedNavOption[];
  size?: SegmentedNavSize;
}) {
  return (
    <nav role="tablist" aria-label={ariaLabel} className={segmentedNavClassName(size)}>
      {options.map((option) => {
        const isActive = value === option.value;

        return (
          <button
            key={option.value}
            type="button"
            role="tab"
            aria-selected={isActive}
            className={segmentedNavTabClassName(isActive, { size })}
            onClick={(event) => {
              event.stopPropagation();
              onValueChange(option.value);
            }}
          >
            {option.label}
          </button>
        );
      })}
    </nav>
  );
}
