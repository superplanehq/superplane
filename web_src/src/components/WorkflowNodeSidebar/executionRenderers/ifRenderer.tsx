import { registerExecutionRenderer, ExecutionRendererProps } from './registry'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import JsonView from '@uiw/react-json-view'
import { lightTheme } from '@uiw/react-json-view/light'
import { darkTheme } from '@uiw/react-json-view/dark'

// Custom renderer for IF component executions
registerExecutionRenderer('if', {
  renderCustomSections: ({ execution, isDarkMode }: ExecutionRendererProps) => {
    const inputs = execution.inputs
    const outputs = execution.outputs

    // Extract condition information
    const condition = inputs?.condition

    // Determine which branch was taken
    const trueBranch = outputs?.true
    const falseBranch = outputs?.false
    const branchTaken = trueBranch && trueBranch.length > 0 ? 'true' :
                        falseBranch && falseBranch.length > 0 ? 'false' :
                        'none'

    return (
      <div className="space-y-3">
        {/* Condition Section */}
        <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
          Condition
        </div>
        <div className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 p-3 text-xs">
          <div className="space-y-2">
            <div className="flex items-start gap-2">
              <MaterialSymbol
                name={branchTaken === 'true' ? 'check_circle' : branchTaken === 'false' ? 'cancel' : 'help'}
                size="md"
                className={
                  branchTaken === 'true' ? 'text-green-600 dark:text-green-400' :
                  branchTaken === 'false' ? 'text-red-600 dark:text-red-400' :
                  'text-gray-600 dark:text-gray-400'
                }
              />
              <div className="flex-1">
                <div className="font-medium mb-1">
                  Evaluated to: <span className={`font-bold ${
                    branchTaken === 'true' ? 'text-green-600 dark:text-green-400' :
                    branchTaken === 'false' ? 'text-red-600 dark:text-red-400' :
                    'text-gray-600 dark:text-gray-400'
                  }`}>
                    {branchTaken.toUpperCase()}
                  </span>
                </div>
                {condition !== undefined && (
                  <div className="mt-2">
                    <div className="text-gray-500 dark:text-zinc-400 mb-1">Condition Expression:</div>
                    <div className="bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700 p-2 rounded font-mono text-xs">
                      <JsonView
                        value={condition}
                        style={{
                          fontSize: '12px',
                          fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Consolas, "Liberation Mono", Menlo, monospace',
                          backgroundColor: 'transparent',
                          textAlign: 'left',
                          ...(isDarkMode ? darkTheme : lightTheme)
                        }}
                        displayDataTypes={false}
                        displayObjectSize={false}
                        enableClipboard={false}
                        collapsed={1}
                      />
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Branches Section */}
        <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
          Branches
        </div>
        <div className="space-y-2">
          {/* True Branch */}
          <div className={`bg-zinc-50 dark:bg-zinc-800 border p-3 text-xs ${
            branchTaken === 'true'
              ? 'border-green-500 dark:border-green-600 bg-green-50 dark:bg-green-900/20'
              : 'border-gray-200 dark:border-zinc-700'
          }`}>
            <div className="flex items-center gap-2 mb-2">
              <MaterialSymbol
                name="arrow_forward"
                size="md"
                className={branchTaken === 'true' ? 'text-green-600 dark:text-green-400' : 'text-gray-400 dark:text-zinc-500'}
              />
              <span className={`font-semibold uppercase tracking-wide ${
                branchTaken === 'true' ? 'text-green-700 dark:text-green-300' : 'text-gray-600 dark:text-zinc-400'
              }`}>
                TRUE Branch
              </span>
              {branchTaken === 'true' && (
                <span className="ml-auto px-2 py-0.5 rounded text-xs font-medium bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300">
                  Taken
                </span>
              )}
            </div>
            {trueBranch && trueBranch.length > 0 ? (
              <div className="pl-6 text-left">
                <div className="text-gray-500 dark:text-zinc-400 mb-1">
                  Output ({trueBranch.length} {trueBranch.length === 1 ? 'item' : 'items'}):
                </div>
                <JsonView
                  value={trueBranch}
                  style={{
                    fontSize: '12px',
                    fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Consolas, "Liberation Mono", Menlo, monospace',
                    backgroundColor: 'transparent',
                    textAlign: 'left',
                    ...(isDarkMode ? darkTheme : lightTheme)
                  }}
                  displayDataTypes={false}
                  displayObjectSize={false}
                  enableClipboard={false}
                  collapsed={1}
                />
              </div>
            ) : (
              <div className="pl-6 text-gray-400 dark:text-zinc-500 italic">No output</div>
            )}
          </div>

          {/* False Branch */}
          <div className={`bg-zinc-50 dark:bg-zinc-800 border p-3 text-xs ${
            branchTaken === 'false'
              ? 'border-red-500 dark:border-red-600 bg-red-50 dark:bg-red-900/20'
              : 'border-gray-200 dark:border-zinc-700'
          }`}>
            <div className="flex items-center gap-2 mb-2">
              <MaterialSymbol
                name="arrow_forward"
                size="md"
                className={branchTaken === 'false' ? 'text-red-600 dark:text-red-400' : 'text-gray-400 dark:text-zinc-500'}
              />
              <span className={`font-semibold uppercase tracking-wide ${
                branchTaken === 'false' ? 'text-red-700 dark:text-red-300' : 'text-gray-600 dark:text-zinc-400'
              }`}>
                FALSE Branch
              </span>
              {branchTaken === 'false' && (
                <span className="ml-auto px-2 py-0.5 rounded text-xs font-medium bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300">
                  Taken
                </span>
              )}
            </div>
            {falseBranch && falseBranch.length > 0 ? (
              <div className="pl-6 text-left">
                <div className="text-gray-500 dark:text-zinc-400 mb-1">
                  Output ({falseBranch.length} {falseBranch.length === 1 ? 'item' : 'items'}):
                </div>
                <JsonView
                  value={falseBranch}
                  style={{
                    fontSize: '12px',
                    fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Consolas, "Liberation Mono", Menlo, monospace',
                    backgroundColor: 'transparent',
                    textAlign: 'left',
                    ...(isDarkMode ? darkTheme : lightTheme)
                  }}
                  displayDataTypes={false}
                  displayObjectSize={false}
                  enableClipboard={false}
                  collapsed={1}
                />
              </div>
            ) : (
              <div className="pl-6 text-gray-400 dark:text-zinc-500 italic">No output</div>
            )}
          </div>
        </div>

        {/* Input Data Section (if there's more than just condition) */}
        {inputs && Object.keys(inputs).filter(k => k !== 'condition').length > 0 && (
          <>
            <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
              Additional Input Data
            </div>
            <div className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 p-3 text-xs text-left">
              <JsonView
                value={Object.fromEntries(Object.entries(inputs).filter(([k]) => k !== 'condition'))}
                style={{
                  fontSize: '12px',
                  fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Consolas, "Liberation Mono", Menlo, monospace',
                  backgroundColor: 'transparent',
                  textAlign: 'left',
                  ...(isDarkMode ? darkTheme : lightTheme)
                }}
                displayDataTypes={false}
                displayObjectSize={false}
                enableClipboard={false}
                collapsed={1}
              />
            </div>
          </>
        )}
      </div>
    )
  }
})
