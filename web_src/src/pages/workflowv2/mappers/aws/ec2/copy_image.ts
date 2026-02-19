import { ComponentBaseMapper } from "../../types";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { buildEc2OperationMapper } from "./operation_mapper";

interface Configuration {
  region?: string;
  sourceRegion?: string;
  sourceImageId?: string;
  name?: string;
}

interface Output {
  requestId?: string;
  imageId?: string;
  sourceImageId?: string;
  sourceRegion?: string;
  region?: string;
  name?: string;
  description?: string;
  state?: string;
}

export const copyImageMapper: ComponentBaseMapper = buildEc2OperationMapper<Configuration, Output>({
  metadata(configuration): MetadataItem[] {
    const items: MetadataItem[] = [];
    if (configuration?.region) {
      items.push({ icon: "globe", label: `to ${configuration.region}` });
    }
    if (configuration?.sourceRegion) {
      items.push({ icon: "map", label: `from ${configuration.sourceRegion}` });
    }
    if (configuration?.sourceImageId) {
      items.push({ icon: "disc", label: configuration.sourceImageId });
    }
    if (configuration?.name) {
      items.push({ icon: "tag", label: configuration.name });
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
      "Source Image ID": stringOrDash(output.sourceImageId),
      "Source Region": stringOrDash(output.sourceRegion),
      Region: stringOrDash(output.region),
      Name: stringOrDash(output.name),
      Description: stringOrDash(output.description),
      State: stringOrDash(output.state),
    };
  },
});
