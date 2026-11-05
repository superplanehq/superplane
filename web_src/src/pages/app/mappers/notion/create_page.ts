import type { MetadataItem } from "@/ui/metadataList";
import { buildSubtitle } from "../eventDisplay";
import type { ComponentBaseMapper, SubtitleContext } from "../types";
import { notionComponentProps, notionExecutionDetails, notionOutput } from "./base";
import type { CreatePageConfiguration, NotionPage } from "./types";

export const createPageMapper: ComponentBaseMapper = {
  props(context) {
    const configuration = context.node.configuration as CreatePageConfiguration | undefined;
    const metadata: MetadataItem[] = [];
    if (configuration?.title) metadata.push({ icon: "file-text", label: configuration.title });
    if (configuration?.parentId) {
      const parentLabel = configuration.parentType === "data_source" ? "Data source" : "Page";
      metadata.push({ icon: "folder", label: `${parentLabel}: ${configuration.parentId}` });
    }
    return notionComponentProps(context, metadata, createPageSubtitle);
  },

  getExecutionDetails({ execution }) {
    const details = notionExecutionDetails(execution);
    const page = notionOutput<NotionPage>(execution);
    if (page?.id) details["Page ID"] = page.id;
    const title = pageTitle(page);
    if (title) details["Title"] = title;
    if (page?.url) details["Page URL"] = page.url;
    return details;
  },

  subtitle: createPageSubtitle,
};

function createPageSubtitle({ execution }: SubtitleContext) {
  return buildSubtitle(pageTitle(notionOutput<NotionPage>(execution)), execution.createdAt);
}

function pageTitle(page: NotionPage | undefined): string {
  const property = Object.values(page?.properties ?? {}).find((value) => value?.type === "title");
  if (!Array.isArray(property?.title)) return "";
  return property.title.map((text) => text?.plain_text ?? text?.text?.content ?? "").join("");
}
