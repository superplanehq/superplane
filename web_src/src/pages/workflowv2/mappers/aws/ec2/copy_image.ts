import { ComponentBaseMapper } from "../../types";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { buildEc2OperationMapper } from "./operation_mapper";
import { Ec2Image } from "./types";

interface Configuration {
  region?: string;
  sourceRegion?: string;
  sourceImageId?: string;
  name?: string;
}

interface Output {
  image?: Ec2Image;
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
  details(configuration, output): Record<string, string> {
    if (!output) {
      return {
        "Source Image ID": stringOrDash(configuration?.sourceImageId),
      };
    }

    return {
      "Source Image ID": stringOrDash(configuration?.sourceImageId),
      "Image ID": stringOrDash(output.image?.imageId),
      Name: stringOrDash(output.image?.name),
      Description: stringOrDash(output.image?.description),
      State: stringOrDash(output.image?.state),
      "Creation Date": stringOrDash(output.image?.creationDate),
      Architecture: stringOrDash(output.image?.architecture),
      "Image Type": stringOrDash(output.image?.imageType),
      "Root Device Type": stringOrDash(output.image?.rootDeviceType),
      "Root Device Name": stringOrDash(output.image?.rootDeviceName),
      "Virtualization Type": stringOrDash(output.image?.virtualizationType),
      Hypervisor: stringOrDash(output.image?.hypervisor),
      "Owner ID": stringOrDash(output.image?.ownerId),
    };
  },
});
