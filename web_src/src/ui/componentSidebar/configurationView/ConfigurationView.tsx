import { ConfigurationValueDisplay } from "./ConfigurationValueDisplay";
import { parseConfigurationDisplayBlocks } from "./configurationDisplayBlocks";
import { ConfigurationDisplayBlockList } from "./ConfigurationNestedGroup";
import type { ConfigurationDisplayModel, ConfigurationDisplayRow } from "./types";

type ConfigurationViewProps = {
  model: ConfigurationDisplayModel;
};

function ConfigurationRow({ row }: { row: ConfigurationDisplayRow }) {
  return (
    <div className="flex items-start gap-2">
      <span className="w-[120px] shrink-0 truncate text-right text-gray-500" title={row.label}>
        {row.label}:
      </span>
      <ConfigurationValueDisplay row={row} className="min-w-0 break-all text-gray-800" />
    </div>
  );
}

export function ConfigurationView({ model }: ConfigurationViewProps) {
  return (
    <div className="flex flex-col gap-1.5 text-[13px]">
      <ConfigurationDisplayBlockList
        blocks={parseConfigurationDisplayBlocks(model.rows)}
        renderRow={(row) => <ConfigurationRow row={row} />}
        hideHeaderSummaries
      />
    </div>
  );
}
