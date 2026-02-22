import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState, useRef, useEffect, useCallback } from "react";
import Editor from "@monaco-editor/react";
import {
  scriptsListScripts,
  scriptsDescribeScript,
  scriptsCreateScript,
  scriptsUpdateScript,
  scriptsDeleteScript,
  scriptsGenerateScript,
} from "@/api-client/sdk.gen";
import type { ScriptsScript } from "@/api-client/types.gen";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import { FileCode2, Send, Save, Trash2, Bot, User, Loader2, Plus, Rocket } from "lucide-react";

interface ChatMessage {
  role: "user" | "assistant";
  content: string;
}

function statusVariant(status?: string): "default" | "secondary" | "destructive" | "outline" {
  switch (status) {
    case "active":
      return "default";
    case "error":
      return "destructive";
    default:
      return "secondary";
  }
}

interface ScriptBuilderModalProps {
  isOpen: boolean;
  onClose: () => void;
  organizationId: string;
}

export function ScriptBuilderModal({ isOpen, onClose, organizationId }: ScriptBuilderModalProps) {
  const queryClient = useQueryClient();

  const [selectedScriptId, setSelectedScriptId] = useState<string | null>(null);
  const [scriptName, setScriptName] = useState("");
  const [scriptLabel, setScriptLabel] = useState("");
  const [source, setSource] = useState("");
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);
  const [chatInput, setChatInput] = useState("");
  const [isDirty, setIsDirty] = useState(false);
  const chatEndRef = useRef<HTMLDivElement>(null);

  const { data: scriptsList } = useQuery({
    queryKey: ["scripts", organizationId],
    queryFn: async () => {
      const response = await scriptsListScripts(withOrganizationHeader({}));
      return response.data?.scripts || [];
    },
    enabled: isOpen && !!organizationId,
  });

  const { data: script } = useQuery({
    queryKey: ["script", selectedScriptId],
    queryFn: async () => {
      const response = await scriptsDescribeScript(withOrganizationHeader({ path: { id: selectedScriptId! } }));
      return response.data?.script;
    },
    enabled: !!selectedScriptId,
  });

  useEffect(() => {
    if (script) {
      setScriptName(script.name || "");
      setScriptLabel(script.label || "");
      setSource(script.source || "");
      setIsDirty(false);
      setChatMessages([]);
    }
  }, [script]);

  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [chatMessages]);

  useEffect(() => {
    if (!isOpen) {
      setSelectedScriptId(null);
      setScriptName("");
      setScriptLabel("");
      setSource("");
      setChatMessages([]);
      setChatInput("");
      setIsDirty(false);
    }
  }, [isOpen]);

  const createMutation = useMutation({
    mutationFn: async () => {
      const name = `script-${Date.now()}`;
      const response = await scriptsCreateScript(
        withOrganizationHeader({
          body: {
            script: {
              name,
              label: "New Component",
              description: "",
            },
          },
        }),
      );
      return response.data?.script;
    },
    onSuccess: (newScript) => {
      queryClient.invalidateQueries({ queryKey: ["scripts", organizationId] });
      if (newScript?.id) {
        setSelectedScriptId(newScript.id);
      }
    },
    onError: (error) => {
      toast.error(`Failed to create component: ${error.message}`);
    },
  });

  const updateMutation = useMutation({
    mutationFn: async () => {
      await scriptsUpdateScript(
        withOrganizationHeader({
          path: { id: selectedScriptId! },
          body: {
            script: {
              name: scriptName,
              label: scriptLabel,
              source,
            },
          },
        }),
      );
    },
    onSuccess: () => {
      setIsDirty(false);
      queryClient.invalidateQueries({ queryKey: ["script", selectedScriptId] });
      queryClient.invalidateQueries({ queryKey: ["scripts", organizationId] });
      toast.success("Component saved");
    },
    onError: () => {
      toast.error("Failed to save component");
    },
  });

  const deployMutation = useMutation({
    mutationFn: async () => {
      await scriptsUpdateScript(
        withOrganizationHeader({
          path: { id: selectedScriptId! },
          body: {
            script: {
              name: scriptName,
              label: scriptLabel,
              source,
              status: "active",
            },
          },
        }),
      );
    },
    onSuccess: () => {
      setIsDirty(false);
      queryClient.invalidateQueries({ queryKey: ["scripts", organizationId] });
      queryClient.invalidateQueries({ queryKey: ["components"] });
      queryClient.invalidateQueries({ queryKey: ["triggers"] });
      toast.success("Component deployed");
      onClose();
    },
    onError: (error) => {
      toast.error(`Failed to deploy component: ${error.message}`);
    },
  });

  const deactivateMutation = useMutation({
    mutationFn: async () => {
      await scriptsUpdateScript(
        withOrganizationHeader({
          path: { id: selectedScriptId! },
          body: {
            script: {
              status: "draft",
            },
          },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["script", selectedScriptId] });
      queryClient.invalidateQueries({ queryKey: ["scripts", organizationId] });
      toast.success("Component deactivated");
    },
    onError: (error) => {
      toast.error(`Failed to deactivate: ${error.message}`);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async () => {
      await scriptsDeleteScript(withOrganizationHeader({ path: { id: selectedScriptId! } }));
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["scripts", organizationId] });
      setSelectedScriptId(null);
      setScriptName("");
      setScriptLabel("");
      setSource("");
      setChatMessages([]);
      toast.success("Component deleted");
    },
  });

  const generateMutation = useMutation({
    mutationFn: async (message: string) => {
      const response = await scriptsGenerateScript(
        withOrganizationHeader({
          path: { scriptId: selectedScriptId! },
          body: { message },
        }),
      );
      return response.data;
    },
    onSuccess: (data) => {
      if (data?.response) {
        setChatMessages((prev) => [...prev, { role: "assistant", content: data.response! }]);
      }
      if (data?.source) {
        setSource(data.source!);
        setIsDirty(true);
      }
    },
    onError: () => {
      setChatMessages((prev) => [
        ...prev,
        { role: "assistant", content: "Sorry, AI generation failed. Please try again." },
      ]);
    },
  });

  const handleSendChat = useCallback(() => {
    const message = chatInput.trim();
    if (!message || generateMutation.isPending) return;

    setChatMessages((prev) => [...prev, { role: "user", content: message }]);
    setChatInput("");
    generateMutation.mutate(message);
  }, [chatInput, generateMutation]);

  const handleSourceChange = useCallback((value: string | undefined) => {
    setSource(value || "");
    setIsDirty(true);
  }, []);

  const scripts = scriptsList || [];

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        showCloseButton={true}
        className="max-w-[90vw] sm:max-w-[90vw] w-[90vw] h-[90vh] p-0 gap-0 flex flex-col overflow-hidden"
      >
        <div className="flex flex-1 min-h-0">
          {/* Left sidebar - Script list */}
          <div className="w-56 border-r border-gray-200 dark:border-gray-700 flex flex-col flex-shrink-0">
            <div className="p-3 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
              <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">Custom Components</h3>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => createMutation.mutate()}
                disabled={createMutation.isPending}
              >
                <Plus className="h-4 w-4" />
              </Button>
            </div>
            <div className="flex-1 overflow-y-auto p-2 space-y-1">
              {scripts.length === 0 && (
                <div className="text-center py-8 px-2">
                  <FileCode2 className="h-8 w-8 mx-auto text-gray-300 dark:text-gray-600 mb-2" />
                  <p className="text-xs text-gray-500 dark:text-gray-400">No custom components yet</p>
                </div>
              )}
              {scripts.map((s: ScriptsScript) => (
                <button
                  key={s.id}
                  onClick={() => setSelectedScriptId(s.id!)}
                  className={`w-full text-left px-3 py-2 rounded-md text-sm transition-colors ${
                    s.id === selectedScriptId
                      ? "bg-gray-100 dark:bg-gray-700 text-gray-900 dark:text-gray-100 font-medium"
                      : "text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-700/50"
                  }`}
                >
                  <div className="flex items-center gap-2">
                    <FileCode2 className="h-3.5 w-3.5 flex-shrink-0" />
                    <span className="truncate">{s.label || s.name}</span>
                  </div>
                </button>
              ))}
            </div>
          </div>

          {selectedScriptId ? (
            <>
              {/* Center panel - AI Chat */}
              <div className="flex-1 flex flex-col border-r border-gray-200 dark:border-gray-700 min-w-0">
                <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700">
                  <h2 className="text-sm font-medium text-gray-900 dark:text-gray-100">AI Assistant</h2>
                  <p className="text-xs text-gray-500 dark:text-gray-400">Describe what you want the component to do</p>
                </div>

                <div className="flex-1 overflow-y-auto p-4 space-y-4">
                  {chatMessages.length === 0 && (
                    <div className="text-center py-12 text-gray-400 dark:text-gray-500">
                      <Bot className="h-10 w-10 mx-auto mb-3 opacity-50" />
                      <p className="text-sm">
                        Describe what you need and the AI will generate TypeScript code using the @superplane/sdk API.
                      </p>
                    </div>
                  )}
                  {chatMessages.map((msg, i) => (
                    <div key={i} className="flex gap-3">
                      <div
                        className={`flex-shrink-0 w-7 h-7 rounded-full flex items-center justify-center ${
                          msg.role === "user"
                            ? "bg-blue-100 dark:bg-blue-900/30"
                            : "bg-purple-100 dark:bg-purple-900/30"
                        }`}
                      >
                        {msg.role === "user" ? (
                          <User className="h-3.5 w-3.5 text-blue-600 dark:text-blue-400" />
                        ) : (
                          <Bot className="h-3.5 w-3.5 text-purple-600 dark:text-purple-400" />
                        )}
                      </div>
                      <div className="flex-1 min-w-0">
                        <pre className="whitespace-pre-wrap text-sm text-gray-700 dark:text-gray-300 font-sans">
                          {msg.content}
                        </pre>
                      </div>
                    </div>
                  ))}
                  {generateMutation.isPending && (
                    <div className="flex gap-3">
                      <div className="flex-shrink-0 w-7 h-7 rounded-full flex items-center justify-center bg-purple-100 dark:bg-purple-900/30">
                        <Loader2 className="h-3.5 w-3.5 text-purple-600 dark:text-purple-400 animate-spin" />
                      </div>
                      <div className="text-sm text-gray-500 dark:text-gray-400 pt-1">Generating...</div>
                    </div>
                  )}
                  <div ref={chatEndRef} />
                </div>

                <div className="p-3 border-t border-gray-200 dark:border-gray-700">
                  <div className="flex gap-2">
                    <Input
                      value={chatInput}
                      onChange={(e) => setChatInput(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter" && !e.shiftKey) {
                          e.preventDefault();
                          handleSendChat();
                        }
                      }}
                      placeholder="Describe what you need..."
                      className="flex-1"
                    />
                    <Button
                      size="sm"
                      onClick={handleSendChat}
                      disabled={!chatInput.trim() || generateMutation.isPending}
                    >
                      <Send className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </div>

              {/* Right panel - Code Editor */}
              <div className="w-[45%] flex flex-col min-w-0 flex-shrink-0">
                <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700 space-y-2">
                  <div className="flex items-center gap-2">
                    <Input
                      value={scriptLabel}
                      onChange={(e) => {
                        setScriptLabel(e.target.value);
                        setIsDirty(true);
                      }}
                      placeholder="Component label"
                      className="flex-1 text-sm"
                    />
                    <Badge variant={statusVariant(script?.status)}>{script?.status || "draft"}</Badge>
                  </div>
                  <div className="flex items-center gap-2">
                    <Input
                      value={scriptName}
                      onChange={(e) => {
                        setScriptName(e.target.value);
                        setIsDirty(true);
                      }}
                      placeholder="component-name (used as identifier)"
                      className="flex-1 text-xs font-mono"
                    />
                  </div>
                </div>

                <div className="flex-1 min-h-0">
                  <Editor
                    height="100%"
                    defaultLanguage="typescript"
                    value={source}
                    onChange={handleSourceChange}
                    theme="vs-light"
                    options={{
                      minimap: { enabled: false },
                      fontSize: 13,
                      lineNumbers: "on",
                      wordWrap: "on",
                      scrollBeyondLastLine: false,
                      automaticLayout: true,
                      tabSize: 2,
                      padding: { top: 12 },
                    }}
                  />
                </div>

                <div className="px-4 py-3 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between">
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => {
                      if (window.confirm("Are you sure you want to delete this component?")) {
                        deleteMutation.mutate();
                      }
                    }}
                  >
                    <Trash2 className="h-4 w-4 mr-1.5" />
                    Delete
                  </Button>
                  <div className="flex items-center gap-2">
                    {script?.status === "active" && (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => deactivateMutation.mutate()}
                        disabled={deactivateMutation.isPending}
                      >
                        {deactivateMutation.isPending ? "Deactivating..." : "Deactivate"}
                      </Button>
                    )}
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => updateMutation.mutate()}
                      disabled={!isDirty || updateMutation.isPending}
                    >
                      <Save className="h-4 w-4 mr-1.5" />
                      {updateMutation.isPending ? "Saving..." : "Save"}
                    </Button>
                    <Button
                      size="sm"
                      onClick={() => deployMutation.mutate()}
                      disabled={deployMutation.isPending || (!source.trim() && !isDirty)}
                    >
                      <Rocket className="h-4 w-4 mr-1.5" />
                      {deployMutation.isPending ? "Deploying..." : "Deploy"}
                    </Button>
                  </div>
                </div>
              </div>
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-gray-400 dark:text-gray-500">
              <div className="text-center">
                <FileCode2 className="h-12 w-12 mx-auto mb-3 opacity-50" />
                <p className="text-sm mb-4">
                  {scripts.length === 0
                    ? "Create your first custom component."
                    : "Select a component from the sidebar to edit it."}
                </p>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => createMutation.mutate()}
                  disabled={createMutation.isPending}
                >
                  <Plus className="h-4 w-4 mr-1.5" />
                  New Component
                </Button>
              </div>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
