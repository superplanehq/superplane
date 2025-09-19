import Tippy from '@tippyjs/react/headless';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { CodeBlock } from '@/components/CodeBlock/code-block';

interface OutputsHelpTooltipProps {
  className?: string;
  executorType?: string;
}

export function OutputsHelpTooltip({ className = '', executorType }: OutputsHelpTooltipProps) {
  const renderSemaphoreSection = () => (
    <div className="mb-6">
      <p className="text-xs mb-3 text-zinc-600 dark:text-zinc-400">
        From a Semaphore pipeline:
      </p>
      <CodeBlock>
        {`# Add to your .semaphore/semaphore.yml
- name: "Push outputs to Superplane"
  commands:
    - |
      curl \\
        "https://app.superplane.com/api/v1/outputs" \\
        -X POST \\
        -H "Content-Type: application/json" \\
        -H "Authorization: Bearer $SEMAPHORE_OIDC_TOKEN" \\
        --data '{
          "execution_id": "'$SUPERPLANE_STAGE_EXECUTION_ID'",
          "external_id": "'$SEMAPHORE_WORKFLOW_ID'",
          "outputs": {
            "BUILD_URL": "https://builds.example.com/123",
            "TEST_COVERAGE": "87%",
            "IMAGE_TAG": "my-app:v1.2.3"
          }
        }'`}
      </CodeBlock>
      <p className="text-xs text-zinc-500 dark:text-zinc-500">
        <code>SEMAPHORE_OIDC_TOKEN</code> and <code>SUPERPLANE_STAGE_EXECUTION_ID</code> are automatically available in your Semaphore environment.
      </p>
    </div>
  );

  const renderGitHubSection = () => (
    <div className="mb-6">
      <p className="text-xs mb-3 text-zinc-600 dark:text-zinc-400">
        From a GitHub workflow:
      </p>
      <CodeBlock>
        {`# Add to your .github/workflows/*.yml
permissions:
  id-token: write

jobs:
  your-job:
    runs-on: ubuntu-latest
    steps:
      - name: Install OIDC Client
        run: npm install @actions/core@1.6.0 @actions/http-client

      - name: Get Id Token
        uses: actions/github-script@v7
        id: idToken
        with:
          script: |
            let token = await core.getIDToken('superplane')
            core.setOutput('token', token)

      - name: Push outputs to Superplane
        run: |
          curl \\
            "https://app.superplane.com/api/v1/outputs" \\
            -X POST \\
            -H "Content-Type: application/json" \\
            -H "Authorization: Bearer $GITHUB_ID_TOKEN" \\
            --data '{
              "execution_id": "'$SUPERPLANE_EXECUTION_ID'",
              "external_id": "'$GITHUB_RUN_ID'",
              "outputs": {
                "DEPLOY_URL": "https://app-staging.example.com",
                "HEALTH_CHECK": "passing"
              }
            }'
        env:
          GITHUB_ID_TOKEN: \${{ steps.idToken.outputs.token }}
          SUPERPLANE_EXECUTION_ID: \${{ inputs.superplane_execution_id }}`}
      </CodeBlock>
      <p className="text-xs text-zinc-500 dark:text-zinc-500">
        <code>SUPERPLANE_EXECUTION_ID</code> is passed as an input to your GitHub workflow.
      </p>
    </div>
  );

  const renderHttpSection = () => (
    <div className="mb-6">
      <p className="text-xs mb-4 text-zinc-600 dark:text-zinc-400">
        Outputs are key-value pairs that stages produce during execution. Downstream stages can then access these outputs as inputs. Here’s how to push outputs from your executions:
      </p>
      <CodeBlock>
        {`{
  "status": "success",
  "message": "Deployment completed",
  "outputs": {
    "DEPLOY_URL": "https://app-staging.example.com",
    "BUILD_VERSION": "v1.2.3",
    "DEPLOY_TIME": "2024-01-15T10:30:00Z"
  }
}`}
      </CodeBlock>
      <p className="text-xs text-zinc-500 dark:text-zinc-500">
        Superplane will automatically extract the <code>outputs</code> field from your HTTP response.
      </p>
    </div>
  );

  const renderParametersSection = () => (
    <div className="border-t border-zinc-200 dark:border-zinc-700 pt-4">
      <h4 className="text-xs font-medium mb-2 text-zinc-700 dark:text-zinc-300">
        {executorType === 'http' ? 'Response Format:' : 'API Parameters:'}
      </h4>
      <ul className="text-xs space-y-1 text-zinc-600 dark:text-zinc-400">
        {executorType !== 'http' && (
          <>
            <li><code className="bg-zinc-100 dark:bg-zinc-800 px-1 rounded">execution_id</code> - The ID of the Superplane stage execution.</li>
            <li><code className="bg-zinc-100 dark:bg-zinc-800 px-1 rounded">external_id</code> - The unique ID from the external system (e.g., <code>SEMAPHORE_WORKFLOW_ID</code>).</li>
          </>
        )}
        <li><code className="bg-zinc-100 dark:bg-zinc-800 px-1 rounded">outputs</code> - A JSON object of key-value pairs to be stored as outputs.</li>
      </ul>
    </div>
  );

  const tooltipContent = (
    <div className="nodrag">
      <p className="text-xs mb-4 text-zinc-600 dark:text-zinc-400">
        Outputs are key-value pairs that stages produce during execution. Downstream stages can then access these outputs as inputs. Here’s how to push outputs from your executions:
      </p>

      {/* Render specific section based on executor type */}
      {executorType === 'semaphore' && renderSemaphoreSection()}
      {executorType === 'github' && renderGitHubSection()}
      {executorType === 'http' && renderHttpSection()}

      {/* If no specific executor type, show all sections */}
      {!executorType && (
        <>
          {renderSemaphoreSection()}
          {renderGitHubSection()}
          {renderHttpSection()}
        </>
      )}

      {renderParametersSection()}
    </div>
  );

  return (
    <Tippy
      render={() => (
        <div onClick={(e) => e.stopPropagation()} className="max-w-[800px]">
          <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-4 text-sm z-50 font-normal">
            <div className="font-semibold mb-3 text-zinc-900 dark:text-zinc-100">How to Push Outputs</div>
            {tooltipContent}
          </div>
        </div>
      )}
      placement="top"
      interactive
      delay={200}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        className={`text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300 transition-colors cursor-pointer ${className}`}
        title="How to push outputs"
        role="button"
        tabIndex={0}
      >
        <MaterialSymbol name="help" size="sm" />
      </div>
    </Tippy>
  );
}