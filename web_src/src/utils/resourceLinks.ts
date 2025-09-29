/**
 * Utility functions for generating external resource links for integrations
 */

export interface ResourceLinkConfig {
  url: string;
  tooltip: string;
}

/**
 * Generate link to Semaphore project
 */
export function getSemaphoreProjectLink(integrationUrl: string, projectName: string, ref?: string): ResourceLinkConfig | null {
  if (!integrationUrl || !projectName) return null;

  const cleanProjectName = projectName.replace('.semaphore/', '');
  const baseUrl = integrationUrl.endsWith('/') ? integrationUrl.slice(0, -1) : integrationUrl;

  let url = `${baseUrl}/projects/${cleanProjectName}`;
  if (ref) {
    if (ref.startsWith('refs/tags/')) {
      url += `?type=tag`;
    } else {
      url += `?type=branch`;
    }
  }
  const tooltip = ref
    ? `Open ${cleanProjectName} project at ${ref} in Semaphore`
    : `Open ${cleanProjectName} project in Semaphore`;

  return {
    url,
    tooltip
  };
}

/**
 * Generate link to Semaphore pipeline file on GitHub
 */
export function getSemaphorePipelineLink(
  integrationUrl: string,
  projectName: string,
): ResourceLinkConfig | null {
  if (!integrationUrl || !projectName) return null;

  const githubUrl = `${integrationUrl}/projects/${projectName}/edit_workflow`;

  return {
    url: githubUrl,
    tooltip: `View ${projectName} pipeline file`
  };
}

/**
 * Generate link to GitHub repository
 */
export function getGitHubRepositoryLink(integrationUrl: string, repositoryName: string, ref?: string): ResourceLinkConfig | null {
  if (!integrationUrl || !repositoryName) return null;

  const baseUrl = `${integrationUrl}/${repositoryName}`;
  const url = ref ? `${baseUrl}/tree/${ref}` : baseUrl;
  const tooltip = ref
    ? `Open ${repositoryName} repository at ${ref} on GitHub`
    : `Open ${repositoryName} repository on GitHub`;

  return {
    url,
    tooltip
  };
}

/**
 * Generate link to GitHub Actions workflow file
 */
export function getGitHubWorkflowLink(
  integrationUrl: string,
  repositoryName: string,
  workflowFile: string,
  ref?: string
): ResourceLinkConfig | null {
  if (!integrationUrl || !repositoryName || !workflowFile) return null;

  const cleanWorkflowFile = workflowFile.replace('.github/workflows/', '');
  const gitRef = ref || 'main';

  return {
    url: `${integrationUrl}/${repositoryName}/blob/${gitRef}/.github/workflows/${cleanWorkflowFile}`,
    tooltip: `View ${cleanWorkflowFile} workflow file on GitHub`
  };
}

/**
 * Generic function to get resource links based on integration type and executor spec
 */
export function getResourceLinks(
  integrationType: string,
  integrationUrl?: string,
  resourceName?: string,
  executorSpec?: Record<string, unknown>
): ResourceLinkConfig[] {
  const links: ResourceLinkConfig[] = [];

  if (!integrationType || !integrationUrl) return links;

  if (!resourceName) {
    const pureIntegrationUrlLink = {
      url: integrationUrl,
      tooltip: `Open ${integrationUrl}`
    }
    links.push(pureIntegrationUrlLink);
    return links;
  }

  switch (integrationType) {
    case 'semaphore': {
      const ref = executorSpec?.ref as string | undefined;

      if (integrationUrl) {
        const projectLink = getSemaphoreProjectLink(integrationUrl, resourceName);
        if (projectLink) links.push(projectLink);

        const pipelineLink = getSemaphorePipelineLink(integrationUrl, resourceName);
        if (pipelineLink) links.push(pipelineLink);

        if (ref) {
          const projectLink = getSemaphoreProjectLink(integrationUrl, resourceName, ref);
          if (projectLink) links.push(projectLink);
        }
      }
      break;
    }

    case 'github': {
      const repoLink = getGitHubRepositoryLink(integrationUrl, resourceName);
      if (repoLink) links.push(repoLink);

      if (executorSpec?.workflow) {
        const workflowLink = getGitHubWorkflowLink(
          integrationUrl,
          resourceName,
          executorSpec.workflow as string,
          executorSpec.ref as string | undefined
        );
        if (workflowLink) links.push(workflowLink);

      }

      if (executorSpec?.ref) {
        const branchLink = getGitHubRepositoryLink(integrationUrl, resourceName, executorSpec.ref as string );
        if (branchLink) links.push(branchLink);
      }
      
      break;
    }
  }

  return links;
}