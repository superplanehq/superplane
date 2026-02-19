import { ComponentBaseMapper } from "../../types";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { buildEc2OperationMapper } from "./operation_mapper";

interface Configuration {
  region?: string;
  imageId?: string;
}

interface Output {
  requestId?: string;
  imageId?: string;
  region?: string;
  disabled?: boolean;
}

export const disableImageMapper: ComponentBaseMapper = buildEc2OperationMapper<Configuration, Output>({
  metadata(configuration): MetadataItem[] {
    const items: MetadataItem[] = [];
    if (configuration?.region) {
      items.push({ icon: "globe", label: configuration.region });
    }
    if (configuration?.imageId) {
      items.push({ icon: "disc", label: configuration.imageId });
    }
    return items;
  },
  details(_configuration, output): Record<string, string> {
    if (!output) {
      return {};
    }

    return {
      "Request ID": stringOrDash(output.requestId),
      "Image ID": stringOrDash(output.imageId),
      Region: stringOrDash(output.region),
      Disabled: stringOrDash(output.disabled),
    };
  },
});
