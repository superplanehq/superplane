import { PersistentTooltip } from './persistent-tooltip';
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
        Add this to your Semaphore pipeline to push outputs using OIDC authentication:
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
        Add these steps to your GitHub workflow to push outputs using OIDC. The GITHUB_ID_TOKEN must be generated in a previous step, like in the example below:
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
    </div>
  );

  const renderHttpSection = () => (
    <div className="mb-6">
      <p className="text-xs mb-3 text-zinc-600 dark:text-zinc-400">
        For HTTP executors, return outputs in your endpoint's JSON response:
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
            <li><code className="bg-zinc-100 dark:bg-zinc-800 px-1 rounded">execution_id</code> - Superplane execution identifier (passed as parameter)</li>
            <li><code className="bg-zinc-100 dark:bg-zinc-800 px-1 rounded">external_id</code> - External system's run/workflow identifier</li>
          </>
        )}
        <li><code className="bg-zinc-100 dark:bg-zinc-800 px-1 rounded">outputs</code> - Key-value pairs of output data</li>
      </ul>
    </div>
  );

  const tooltipContent = (
    <div className="nodrag">
      <p className="text-xs mb-4 text-zinc-600 dark:text-zinc-400">
        {executorType === 'http'
          ? 'Return outputs in your HTTP endpoint\'s JSON response body.'
          : 'Push outputs back to Superplane using the /outputs API endpoint.'
        }
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
    <PersistentTooltip
      content={tooltipContent}
      title="Pushing Outputs from Executions"
      maxWidth={800}
      maxHeight=""
      className={className}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300 transition-colors cursor-pointer"
        title="How to push outputs"
        role="button"
        tabIndex={0}
      >
        <MaterialSymbol name="help" size="sm" />
      </div>
    </PersistentTooltip>
  );
}