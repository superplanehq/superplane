import { Sparkles } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Heading } from "@/components/Heading/heading";

export function AgentPanel() {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-center">
      <div className="rounded-2xl bg-gradient-to-br from-violet-100 to-purple-50 dark:from-violet-900/30 dark:to-purple-900/20 p-4 mb-4">
        <Sparkles size={28} className="text-violet-500 dark:text-violet-400" />
      </div>
      <Heading level={3} className="!text-lg mb-2">
        AI-powered Canvas builder
      </Heading>
      <p className="text-[13px] text-gray-500 dark:text-gray-400 max-w-sm leading-relaxed mb-3">
        Describe what you want in plain English and let the agent build your Canvas. Connect triggers, configure
        components, and wire everything up from a single prompt.
      </p>
      <Badge variant="secondary" className="text-xs">
        Coming soon
      </Badge>
    </div>
  );
}
