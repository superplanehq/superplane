import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useCanvasTemplates } from "@/hooks/useCanvasData";
import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { TemplateCard } from "./CreateCanvasPage";
import { ALL_TAGS, getTemplateTags } from "./templateMetadata";
import { Link, useParams } from "react-router-dom";
import { useMemo, useState } from "react";
import { ArrowLeft, LayoutTemplate, Search } from "lucide-react";
import { Input } from "@/components/Input/input";
import type { CanvasesCanvas } from "@/api-client";

export function TemplatesPage() {
  usePageTitle(["Templates"]);
  const { organizationId } = useParams<{ organizationId: string }>();
  const { data: templates = [], isLoading } = useCanvasTemplates(organizationId || "");
  const [activeTag, setActiveTag] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");

  const filteredTemplates = useMemo(() => {
    let result = templates;

    if (activeTag) {
      result = result.filter((t: CanvasesCanvas) => getTemplateTags(t.metadata?.name).includes(activeTag));
    }

    const query = searchQuery.trim().toLowerCase();
    if (query) {
      result = result.filter(
        (t: CanvasesCanvas) =>
          t.metadata?.name?.toLowerCase().includes(query) || t.metadata?.description?.toLowerCase().includes(query),
      );
    }

    return result;
  }, [templates, activeTag, searchQuery]);

  return (
    <div className="min-h-screen flex flex-col bg-slate-100 dark:bg-gray-900">
      <header className="bg-white dark:bg-gray-900 border-b border-slate-950/15 dark:border-gray-800 px-4 h-12 flex items-center gap-4">
        <OrganizationMenuButton organizationId={organizationId || ""} />
        <Link
          to={`/${organizationId}`}
          className="flex items-center gap-1.5 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200 transition-colors"
        >
          <ArrowLeft size={16} />
          <span>Back to Canvases</span>
        </Link>
      </header>
      <main className="w-full h-full flex flex-column flex-grow-1">
        <div className="w-full flex-grow-1">
          <div className="p-8 max-w-5xl mx-auto">
            <div className="mb-6">
              <Heading level={2} className="!text-2xl mb-1">
                Templates
              </Heading>
              <Text className="text-gray-800 dark:text-gray-400">
                Pre-built workflows to get you started. Pick one to preview and customize.
              </Text>
            </div>

            {!isLoading && templates.length > 0 ? (
              <div className="flex flex-col gap-4 mb-6">
                <div className="relative max-w-xs">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={16} />
                  <Input
                    placeholder="Search templates..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="pl-9"
                  />
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  <Button
                    variant={activeTag === null ? "default" : "outline"}
                    size="sm"
                    onClick={() => setActiveTag(null)}
                    className="h-7 text-xs"
                  >
                    All
                  </Button>
                  {ALL_TAGS.map((tag) => (
                    <Button
                      key={tag}
                      variant={activeTag === tag ? "default" : "outline"}
                      size="sm"
                      onClick={() => setActiveTag(activeTag === tag ? null : tag)}
                      className="h-7 text-xs"
                    >
                      {tag}
                    </Button>
                  ))}
                </div>
              </div>
            ) : null}

            {isLoading ? (
              <div className="flex justify-center items-center h-40">
                <div className="animate-spin rounded-full h-8 w-8 border-b border-blue-600"></div>
                <p className="ml-3 text-gray-500">Loading templates...</p>
              </div>
            ) : templates.length === 0 ? (
              <div className="text-center py-16">
                <LayoutTemplate className="mx-auto text-gray-400 mb-4" size={48} />
                <Heading level={3} className="text-lg text-gray-800 dark:text-white mb-2">
                  No templates available
                </Heading>
                <Text className="text-gray-500 dark:text-gray-400">
                  Templates will appear here once they are configured for your organization.
                </Text>
              </div>
            ) : filteredTemplates.length === 0 ? (
              <div className="text-center py-16">
                <LayoutTemplate className="mx-auto text-gray-400 mb-4" size={48} />
                <Heading level={3} className="text-lg text-gray-800 dark:text-white mb-2">
                  No templates found
                </Heading>
                <Text className="text-gray-500 dark:text-gray-400">
                  Try a different search term or category filter.
                </Text>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {filteredTemplates.map((template: CanvasesCanvas) => (
                  <TemplateCard
                    key={template.metadata?.id}
                    template={template}
                    organizationId={organizationId || ""}
                    showTags
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  );
}
