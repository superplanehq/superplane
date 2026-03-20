import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useCanvasTemplates } from "@/hooks/useCanvasData";
import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { TemplateCard } from "./CreateCanvasPage";
import { useParams } from "react-router-dom";
import { LayoutTemplate } from "lucide-react";

export function TemplatesPage() {
  usePageTitle(["Templates"]);
  const { organizationId } = useParams<{ organizationId: string }>();
  const { data: templates = [], isLoading } = useCanvasTemplates(organizationId || "");

  return (
    <div className="min-h-screen flex flex-col bg-slate-100 dark:bg-gray-900">
      <header className="bg-white border-b border-slate-950/15 px-4 h-12 flex items-center">
        <OrganizationMenuButton organizationId={organizationId || ""} />
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
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {templates.map((template: any) => (
                  <TemplateCard key={template.metadata?.id} template={template} organizationId={organizationId || ""} />
                ))}
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  );
}
