import { usePageTitle } from "@/hooks/usePageTitle";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { ArrowLeft, BotIcon, Code, Loader2, Plus, Save, Send, Trash2 } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

interface JSComponent {
  name: string;
  label: string;
  source: string;
}

interface ChatMessage {
  id: string;
  role: "user" | "assistant";
  content: string;
}

async function apiFetch(path: string, options: RequestInit = {}) {
  const opts = withOrganizationHeader(options);
  const response = await fetch(`/api/v1/js-components${path}`, {
    ...opts,
    headers: {
      "Content-Type": "application/json",
      ...opts.headers,
    },
  });
  return response;
}

export function JSComponentsPage() {
  usePageTitle(["Component Builder"]);
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();

  const [components, setComponents] = useState<JSComponent[]>([]);
  const [selectedComponent, setSelectedComponent] = useState<string | null>(null);
  const [source, setSource] = useState("");
  const [componentName, setComponentName] = useState("");
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputMessage, setInputMessage] = useState("");
  const [isGenerating, setIsGenerating] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const loadComponents = useCallback(async () => {
    try {
      const resp = await apiFetch("");
      if (resp.ok) {
        const data = await resp.json();
        setComponents(data.components || []);
      }
    } catch {
      // ignore
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    loadComponents();
  }, [loadComponents]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const selectComponent = useCallback(
    (name: string) => {
      const comp = components.find((c) => c.name === name);
      if (comp) {
        setSelectedComponent(name);
        setSource(comp.source);
        setComponentName(name);
        setMessages([
          {
            id: "system-1",
            role: "assistant",
            content: `Editing **${comp.label}** (\`${name}\`). Describe the changes you'd like to make.`,
          },
        ]);
      }
    },
    [components],
  );

  const startNew = useCallback(() => {
    setSelectedComponent(null);
    setSource("");
    setComponentName("");
    setMessages([
      {
        id: "welcome",
        role: "assistant",
        content:
          "Describe the component you want to build. For example:\n\n" +
          '- "Create a component that sends a Slack message"\n' +
          '- "Build a component that creates a GitHub issue"\n' +
          '- "Make a component that calls a webhook with JSON data"',
      },
    ]);
  }, []);

  useEffect(() => {
    if (!isLoading && messages.length === 0) {
      startNew();
    }
  }, [isLoading, messages.length, startNew]);

  const extractCodeBlock = (text: string): string | null => {
    const match = text.match(/```(?:javascript|js)?\s*\n([\s\S]*?)```/);
    return match ? match[1].trim() : null;
  };

  const sendMessage = useCallback(async () => {
    const content = inputMessage.trim();
    if (!content || isGenerating) return;

    const userMsg: ChatMessage = {
      id: `user-${Date.now()}`,
      role: "user",
      content,
    };

    setMessages((prev) => [...prev, userMsg]);
    setInputMessage("");
    setIsGenerating(true);

    try {
      const apiMessages = [...messages, userMsg]
        .filter((m) => m.role === "user" || m.role === "assistant")
        .map((m) => ({
          role: m.role === "assistant" ? "assistant" : "user",
          content: m.content,
        }));

      const resp = await apiFetch("/generate", {
        method: "POST",
        body: JSON.stringify({
          messages: apiMessages,
          source: source || undefined,
        }),
      });

      if (!resp.ok) {
        const err = await resp.json();
        throw new Error(err.error || "Generation failed");
      }

      const data = await resp.json();
      const aiContent = data.response || "Sorry, I could not generate a response.";

      const aiMsg: ChatMessage = {
        id: `ai-${Date.now()}`,
        role: "assistant",
        content: aiContent,
      };
      setMessages((prev) => [...prev, aiMsg]);

      const code = extractCodeBlock(aiContent);
      if (code) {
        setSource(code);
        if (!componentName) {
          const labelMatch = code.match(/label:\s*["']([^"']+)["']/);
          if (labelMatch) {
            const suggested = labelMatch[1]
              .toLowerCase()
              .replace(/[^a-z0-9]+/g, "-")
              .replace(/^-|-$/g, "");
            setComponentName(suggested);
          }
        }
      }
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : "Failed to generate";
      setMessages((prev) => [
        ...prev,
        {
          id: `error-${Date.now()}`,
          role: "assistant",
          content: `Error: ${errorMessage}`,
        },
      ]);
    } finally {
      setIsGenerating(false);
    }
  }, [inputMessage, isGenerating, messages, source, componentName]);

  const handleSave = useCallback(async () => {
    if (!componentName.trim() || !source.trim()) {
      showErrorToast("Name and source code are required");
      return;
    }

    setIsSaving(true);
    try {
      const resp = await apiFetch("", {
        method: "POST",
        body: JSON.stringify({
          name: componentName.trim(),
          source: source,
        }),
      });

      if (!resp.ok) {
        const err = await resp.json();
        throw new Error(err.error || "Save failed");
      }

      showSuccessToast("Component saved! It will be available in the sidebar shortly.");
      setSelectedComponent(componentName.trim());
      await loadComponents();
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : "Failed to save";
      showErrorToast(errorMessage);
    } finally {
      setIsSaving(false);
    }
  }, [componentName, source, loadComponents]);

  const handleDelete = useCallback(
    async (name: string) => {
      if (!confirm(`Delete component "${name}"?`)) return;

      try {
        const resp = await apiFetch(`?name=${encodeURIComponent(name)}`, {
          method: "DELETE",
        });

        if (!resp.ok) {
          const err = await resp.json();
          throw new Error(err.error || "Delete failed");
        }

        showSuccessToast("Component deleted");
        if (selectedComponent === name) {
          startNew();
        }
        await loadComponents();
      } catch (err: unknown) {
        const errorMessage = err instanceof Error ? err.message : "Failed to delete";
        showErrorToast(errorMessage);
      }
    },
    [selectedComponent, startNew, loadComponents],
  );

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  return (
    <div className="flex h-screen bg-white">
      {/* Left sidebar - component list */}
      <div className="w-64 border-r border-gray-200 flex flex-col bg-gray-50">
        <div className="p-4 border-b border-gray-200">
          <div className="flex items-center gap-2 mb-3">
            <button
              onClick={() => navigate(`/${organizationId}`)}
              className="p-1 hover:bg-gray-200 rounded transition-colors"
            >
              <ArrowLeft size={16} />
            </button>
            <h1 className="text-sm font-semibold">Component Builder</h1>
          </div>
          <Button variant="outline" size="sm" className="w-full" onClick={startNew}>
            <Plus size={14} />
            New Component
          </Button>
        </div>

        <div className="flex-1 overflow-y-auto p-2">
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 size={16} className="animate-spin text-gray-400" />
            </div>
          ) : components.length === 0 ? (
            <div className="text-center py-8 px-4">
              <Code size={24} className="mx-auto mb-2 text-gray-300" />
              <p className="text-xs text-gray-400">No components yet</p>
            </div>
          ) : (
            components.map((comp) => (
              <div
                key={comp.name}
                className={`group flex items-center justify-between px-3 py-2 rounded cursor-pointer text-sm transition-colors ${
                  selectedComponent === comp.name ? "bg-blue-50 text-blue-700" : "hover:bg-gray-100 text-gray-700"
                }`}
                onClick={() => selectComponent(comp.name)}
              >
                <div className="min-w-0 flex-1">
                  <div className="truncate font-medium">{comp.label}</div>
                  <div className="truncate text-xs text-gray-400">{comp.name}</div>
                </div>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleDelete(comp.name);
                  }}
                  className="opacity-0 group-hover:opacity-100 p-1 hover:bg-red-50 hover:text-red-500 rounded transition-all"
                >
                  <Trash2 size={12} />
                </button>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Main content area */}
      <div className="flex-1 flex">
        {/* Chat panel */}
        <div className="w-[420px] border-r border-gray-200 flex flex-col">
          <div className="px-4 py-3 border-b border-gray-200 flex items-center gap-2">
            <BotIcon size={16} className="text-gray-500" />
            <span className="text-sm font-medium text-gray-700">AI Assistant</span>
          </div>

          {/* Messages */}
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {messages.map((msg) => (
              <div key={msg.id} className={`flex gap-3 ${msg.role === "user" ? "flex-row-reverse" : ""}`}>
                {msg.role === "assistant" && (
                  <div className="flex-shrink-0">
                    <BotIcon size={28} className="text-white bg-gray-900 rounded p-1.5" />
                  </div>
                )}
                <div
                  className={`max-w-[85%] rounded-lg px-3 py-2 text-sm whitespace-pre-wrap ${
                    msg.role === "user" ? "bg-blue-500 text-white" : "bg-gray-100 text-gray-800"
                  }`}
                >
                  {msg.content}
                </div>
              </div>
            ))}

            {isGenerating && (
              <div className="flex gap-3">
                <div className="flex-shrink-0">
                  <BotIcon size={28} className="text-white bg-gray-900 rounded p-1.5" />
                </div>
                <div className="bg-gray-100 rounded-lg px-3 py-2">
                  <Loader2 size={16} className="animate-spin text-gray-400" />
                </div>
              </div>
            )}

            <div ref={messagesEndRef} />
          </div>

          {/* Input */}
          <div className="border-t border-gray-200 p-3">
            <div className="flex gap-2">
              <textarea
                ref={textareaRef}
                value={inputMessage}
                onChange={(e) => setInputMessage(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Describe your component..."
                rows={2}
                className="flex-1 resize-none border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
              <Button
                onClick={sendMessage}
                disabled={!inputMessage.trim() || isGenerating}
                size="sm"
                className="self-end"
              >
                <Send size={14} />
              </Button>
            </div>
          </div>
        </div>

        {/* Code editor panel */}
        <div className="flex-1 flex flex-col">
          <div className="px-4 py-3 border-b border-gray-200 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Code size={16} className="text-gray-500" />
              <span className="text-sm font-medium text-gray-700">Source Code</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="flex items-center gap-1">
                <label className="text-xs text-gray-500">Name:</label>
                <Input
                  value={componentName}
                  onChange={(e) => setComponentName(e.target.value)}
                  placeholder="my-component"
                  className="w-48 h-7 text-xs"
                />
              </div>
              <Button onClick={handleSave} disabled={isSaving || !source.trim() || !componentName.trim()} size="sm">
                {isSaving ? <Loader2 size={14} className="animate-spin" /> : <Save size={14} />}
                Save
              </Button>
            </div>
          </div>

          <div className="flex-1 overflow-hidden">
            <textarea
              value={source}
              onChange={(e) => setSource(e.target.value)}
              spellCheck={false}
              className="w-full h-full p-4 font-mono text-sm bg-gray-950 text-green-400 resize-none focus:outline-none"
              placeholder="// Component source code will appear here after you describe it to the AI..."
            />
          </div>
        </div>
      </div>
    </div>
  );
}
