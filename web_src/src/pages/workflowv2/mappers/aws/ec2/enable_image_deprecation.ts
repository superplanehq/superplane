import { ComponentBaseMapper } from "../../types";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { buildEc2OperationMapper } from "./operation_mapper";

interface Configuration {
  region?: string;
  imageId?: string;
  deprecateAt?: string;
}

interface Output {
  requestId?: string;
  imageId?: string;
  region?: string;
  deprecateAt?: string;
  deprecationEnabled?: boolean;
}

export const enableImageDeprecationMapper: ComponentBaseMapper = buildEc2OperationMapper<
  Configuration,
  Output
>({
  metadata(configuration): MetadataItem[] {
    const items: MetadataItem[] = [];
    if (configuration?.region) {
      items.push({ icon: "globe", label: configuration.region });
    }
    if (configuration?.imageId) {
      items.push({ icon: "disc", label: configuration.imageId });
    }
    if (configuration?.deprecateAt) {
      items.push({ icon: "clock", label: configuration.deprecateAt });
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
      "Deprecate At": stringOrDash(output.deprecateAt),
      "Deprecation Enabled": stringOrDash(output.deprecationEnabled),
    };
  },
});
