import { useState, useCallback } from 'react';
import { createPortal } from 'react-dom';
import Editor from '@monaco-editor/react';
import * as yaml from 'js-yaml';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { Button } from '@/components/Button/button';

interface YamlCodeEditorProps {
  isOpen: boolean;
  onClose: () => void;
  entityType: string;
  entityData: unknown;
  onApply: (updatedData: unknown) => void;
}

export function YamlCodeEditor({
  isOpen,
  onClose,
  entityType,
  entityData,
  onApply
}: YamlCodeEditorProps) {
  const [yamlContent, setYamlContent] = useState(() => {
    try {
      return yaml.dump(entityData, {
        indent: 2,
        lineWidth: -1,
        noRefs: true,
        sortKeys: false
      });
    } catch (error) {
      console.error('Error serializing to YAML:', error);
      return '# Error serializing data to YAML';
    }
  });

  const [parseError, setParseError] = useState<string | null>(null);
  const [isValidYaml, setIsValidYaml] = useState(true);

  const handleEditorChange = useCallback((value: string | undefined) => {
    if (value === undefined) return;

    setYamlContent(value);

    // Validate YAML on change
    try {
      yaml.load(value);
      setParseError(null);
      setIsValidYaml(true);
    } catch (error) {
      setParseError((error as Error).message);
      setIsValidYaml(false);
    }
  }, []);

  const handleApply = useCallback(() => {
    if (!isValidYaml) return;

    try {
      const parsedData = yaml.load(yamlContent);
      onApply(parsedData);
      onClose();
    } catch (error) {
      setParseError((error as Error).message);
      setIsValidYaml(false);
    }
  }, [yamlContent, isValidYaml, onApply, onClose]);

  const handleReset = useCallback(() => {
    try {
      const resetContent = yaml.dump(entityData, {
        indent: 2,
        lineWidth: -1,
        noRefs: true,
        sortKeys: false
      });
      setYamlContent(resetContent);
      setParseError(null);
      setIsValidYaml(true);
    } catch (error) {
      console.error('Error resetting YAML:', error);
    }
  }, [entityData]);

  if (!isOpen) return null;

  return createPortal(
    <div className="fixed inset-0 z-[9999] flex items-center justify-center bg-gray-50/55">
      <div className="bg-white rounded-lg shadow-xl w-[95vw] h-[95vh] max-w-7xl flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-gray-200">
          <div className="flex items-center gap-2">
            <MaterialSymbol name="code" size="md" />
            <h2 className="text-lg font-semibold">
              Edit {entityType} YAML
            </h2>
          </div>
          <div className="flex items-center gap-2">
            <Button
              onClick={handleReset}
              outline
              className="text-sm"
            >
              <MaterialSymbol name="refresh" size="sm" data-slot="icon" />
              Reset
            </Button>
            <button
              onClick={onClose}
              className="p-2 hover:bg-gray-100 rounded-md transition-colors"
              title="Close editor"
            >
              <MaterialSymbol name="close" size="md" />
            </button>
          </div>
        </div>

        {/* Error Display */}
        {parseError && (
          <div className="p-3 bg-red-50 border-b border-red-200">
            <div className="flex items-start gap-2 text-red-800">
              <MaterialSymbol name="error" size="sm" className="mt-0.5 flex-shrink-0" />
              <div>
                <div className="font-medium text-sm">YAML Parse Error</div>
                <div className="text-xs mt-1 font-mono">{parseError}</div>
              </div>
            </div>
          </div>
        )}

        {/* Editor */}
        <div className="flex-1 border-b border-gray-200">
          <Editor
            height="100%"
            defaultLanguage="yaml"
            value={yamlContent}
            onChange={handleEditorChange}
            theme="vs"
            options={{
              minimap: { enabled: false },
              fontSize: 14,
              lineNumbers: 'on',
              rulers: [80],
              wordWrap: 'on',
              folding: true,
              bracketPairColorization: {
                enabled: true
              },
              autoIndent: 'advanced',
              formatOnPaste: true,
              formatOnType: true,
              tabSize: 2,
              insertSpaces: true,
              scrollBeyondLastLine: false,
              renderWhitespace: 'boundary',
              smoothScrolling: true,
              cursorBlinking: 'smooth'
            }}
          />
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between p-4 bg-gray-50">
          <div className="flex items-center gap-2 text-sm text-gray-600">
            <MaterialSymbol name="info" size="sm" />
            <span>
              Edit the YAML configuration for your {entityType}. Changes will be applied to the form when you click Apply.
            </span>
          </div>
          <div className="flex items-center gap-2">
            <Button
              onClick={onClose}
              outline
              className="text-sm"
            >
              Cancel
            </Button>
            <Button
              onClick={handleApply}
              disabled={!isValidYaml}
              color="blue"
              className="text-sm"
            >
              <MaterialSymbol name="check" size="sm" data-slot="icon" />
              Apply Changes
            </Button>
          </div>
        </div>
      </div>
    </div>,
    document.body
  );
}