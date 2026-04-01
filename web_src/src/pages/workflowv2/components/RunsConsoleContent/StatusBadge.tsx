import { getStatusBadgeProps } from "../../canvasRunsUtils";

export function StatusBadge({ status }: { status: string }) {
  const { badgeColor, label } = getStatusBadgeProps(status);
  return (
    <div
      className={`shrink-0 uppercase text-[10px] py-[1.5px] px-[5px] font-semibold rounded flex items-center tracking-wide justify-center text-white ${badgeColor}`}
    >
      <span>{label}</span>
    </div>
  );
}
