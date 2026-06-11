import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ConfigurationValueDisplay } from "./ConfigurationValueDisplay";
import { parseConfigurationDisplayBlocks } from "./configurationDisplayBlocks";
import { ConfigurationDisplayBlockList } from "./ConfigurationNestedGroup";
import type { ConfigurationDisplayModel, ConfigurationDisplayRow } from "./types";

type ConfigurationViewProps = {
  model: ConfigurationDisplayModel;
  onEdit?: () => void;
  editDisabled?: boolean;
  editDisabledTooltip?: string;
};

function ConfigurationRow({ row }: { row: ConfigurationDisplayRow }) {
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-[13px] font-medium text-gray-500">{row.label}</span>
      <ConfigurationValueDisplay row={row} className="text-[13px]" />
    </div>
  );
}

function ConfigurationEditButton({
  onEdit,
  disabled,
  disabledTooltip,
}: {
  onEdit: () => void;
  disabled?: boolean;
  disabledTooltip?: string;
}) {
  const button = (
    <Button type="button" variant="outline" size="sm" onClick={onEdit} disabled={disabled}>
      Edit
    </Button>
  );

  if (disabled && disabledTooltip) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="inline-flex">{button}</div>
        </TooltipTrigger>
        <TooltipContent side="top">{disabledTooltip}</TooltipContent>
      </Tooltip>
    );
  }

  return button;
}

export function ConfigurationView({ model, onEdit, editDisabled, editDisabledTooltip }: ConfigurationViewProps) {
  return (
    <div className="relative bg-slate-100 p-3">
      {onEdit ? (
        <div className="absolute right-3 top-3">
          <ConfigurationEditButton onEdit={onEdit} disabled={editDisabled} disabledTooltip={editDisabledTooltip} />
        </div>
      ) : null}
      <div className="space-y-4">
        <ConfigurationDisplayBlockList
          blocks={parseConfigurationDisplayBlocks(model.rows)}
          renderRow={(row) => <ConfigurationRow row={row} />}
        />
      </div>
    </div>
  );
}
