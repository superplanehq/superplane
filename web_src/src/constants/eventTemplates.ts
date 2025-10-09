import GithubLogo from '@/assets/github-mark.svg';
import SemaphoreLogo from '@/assets/semaphore-logo-sign-black.svg';

export interface EventTemplate {
  name: string;
  description?: string;
  icon?: string;
  image?: string;
  eventType: string;
  nodeType: 'event_source' | 'stage';
  getEventData: () => unknown;
}

export const EVENT_TEMPLATES: EventTemplate[] = [
  {
    name: 'GitHub - Push Event',
    description: 'Event emitted when code is pushed to a GitHub repository',
    image: GithubLogo,
    eventType: 'push',
    nodeType: 'event_source' as const,
    getEventData: () => ({
      ref: "refs/heads/main",
      before: "2364960799e343f8cb594a81b1f34e7219f8254a",
      after: "7fcca06c1b2b2c482df382248610d46cfd789837",
      repository: {
        name: "superplane",
        full_name: "superplanehq/superplane",
        private: false,
        owner: {
          name: "superplanehq",
          email: null,
          login: "superplanehq",
          avatar_url: "https://avatars.githubusercontent.com/u/210748804?v=4",
          gravatar_id: "",
          url: "https://api.github.com/users/superplanehq",
          html_url: "https://github.com/superplanehq",
          type: "Organization",
          user_view_type: "public",
          site_admin: false
        },
        html_url: "https://github.com/superplanehq/superplane",
        description: null,
        fork: false,
        url: "https://api.github.com/repos/superplanehq/superplane",
        created_at: 1746640119,
        updated_at: "2025-09-24T18:47:53Z",
        pushed_at: 1758745245,
        git_url: "git://github.com/superplanehq/superplane.git",
        ssh_url: "git@github.com:superplanehq/superplane.git",
        clone_url: "https://github.com/superplanehq/superplane.git",
        visibility: "public",
        default_branch: "main",
        master_branch: "main",
        organization: "superplanehq",
      },
      pusher: {
        name: "lucaspin",
        email: "lucas@superplane.com"
      },
      organization: {
        login: "superplanehq",
        url: "https://api.github.com/orgs/superplanehq",
        avatar_url: "https://avatars.githubusercontent.com/u/210748804?v=4",
        description: null
      },
      sender: {
        login: "lucaspin",
        avatar_url: "https://avatars.githubusercontent.com/u/12387728?v=4",
        gravatar_id: "",
        url: "https://api.github.com/users/lucaspin",
        html_url: "https://github.com/lucaspin",
        type: "User",
        user_view_type: "public",
        site_admin: false
      },
      created: false,
      deleted: false,
      forced: false,
      base_ref: null,
      compare: "https://github.com/superplanehq/superplane/compare/2364960799e3...7fcca06c1b2b",
      commits: [
        {
          id: "7fcca06c1b2b2c482df382248610d46cfd789837",
          tree_id: "fb692ac9149f575c86374f7cd54e0ba703b03609",
          distinct: true,
          message: "refactor(ui): display only last processed event in event source node (#315)",
          timestamp: "2025-09-24T17:20:44-03:00",
          url: "https://github.com/superplanehq/superplane/commit/7fcca06c1b2b2c482df382248610d46cfd789837",
          author: {
            name: "Lucas Pinheiro",
            email: "lucas@superplane.com",
            username: "lucaspin"
          },
          committer: {
            name: "GitHub",
            email: "noreply@github.com",
            username: "web-flow"
          },
          added: [],
          removed: [
            "pkg/openapi_client/model_status_history.go"
          ],
          modified: [
            "api/swagger/superplane.swagger.json",
            "pkg/grpc/actions/event_sources/create_event_source.go",
            "pkg/grpc/actions/event_sources/describe_event_source.go",
            "pkg/grpc/actions/event_sources/list_event_sources.go",
            "pkg/grpc/actions/events/list_events.go",
            "pkg/grpc/actions/events/list_events_test.go",
            "pkg/grpc/canvas_service.go",
            "pkg/models/event.go",
            "pkg/models/event_source.go",
            "pkg/openapi_client/.openapi-generator/FILES",
            "pkg/openapi_client/api_event.go",
            "pkg/openapi_client/model_superplane_event_source_status.go",
            "pkg/protos/canvases/canvases.pb.go",
            "protos/canvases.proto",
            "web_src/src/api-client/types.gen.ts",
            "web_src/src/hooks/useCanvasData.ts",
            "web_src/src/pages/canvas/components/EventSourceSidebar.tsx",
            "web_src/src/pages/canvas/components/EventStateItem.tsx",
            "web_src/src/pages/canvas/components/nodes/event_source.tsx",
            "web_src/src/pages/canvas/index.tsx"
          ]
        }
      ],
      head_commit: {
        id: "7fcca06c1b2b2c482df382248610d46cfd789837",
        tree_id: "fb692ac9149f575c86374f7cd54e0ba703b03609",
        distinct: true,
        message: "refactor(ui): display only last processed event in event source node (#315)",
        timestamp: "2025-09-24T17:20:44-03:00",
        url: "https://github.com/superplanehq/superplane/commit/7fcca06c1b2b2c482df382248610d46cfd789837",
        author: {
          name: "Lucas Pinheiro",
          email: "lucas@superplane.com",
          username: "lucaspin"
        },
        committer: {
          name: "GitHub",
          email: "noreply@github.com",
          username: "web-flow"
        },
        added: [],
        removed: [
          "pkg/openapi_client/model_status_history.go"
        ],
        modified: [
          "api/swagger/superplane.swagger.json",
          "pkg/grpc/actions/event_sources/create_event_source.go",
          "pkg/grpc/actions/event_sources/describe_event_source.go",
          "pkg/grpc/actions/event_sources/list_event_sources.go",
          "pkg/grpc/actions/events/list_events.go",
          "pkg/grpc/actions/events/list_events_test.go",
          "pkg/grpc/canvas_service.go",
          "pkg/models/event.go",
          "pkg/models/event_source.go",
          "pkg/openapi_client/.openapi-generator/FILES",
          "pkg/openapi_client/api_event.go",
          "pkg/openapi_client/model_superplane_event_source_status.go",
          "pkg/protos/canvases/canvases.pb.go",
          "protos/canvases.proto",
          "web_src/src/api-client/types.gen.ts",
          "web_src/src/hooks/useCanvasData.ts",
          "web_src/src/pages/canvas/components/EventSourceSidebar.tsx",
          "web_src/src/pages/canvas/components/EventStateItem.tsx",
          "web_src/src/pages/canvas/components/nodes/event_source.tsx",
          "web_src/src/pages/canvas/index.tsx"
        ]
      }
    })
  },
  {
    name: 'Semaphore - Pipeline Done Event',
    description: 'Event emitted when a Semaphore CI/CD pipeline completes',
    image: SemaphoreLogo,
    eventType: 'pipeline_done',
    nodeType: 'event_source' as const,
    getEventData: () => ({
      version: "1.0.0",
      organization: {
        name: "superplanehq",
        id: crypto.randomUUID()
      },
      project: {
        name: "superplane",
        id: crypto.randomUUID()
      },
      repository: {
        url: "https://github.com/superplanehq/superplane",
        slug: "superplanehq/superplane"
      },
      revision: {
        sender: {
          login: "lucaspin",
          email: "lucas@superplane.com",
        },
        reference_type: "branch",
        reference: "refs/heads/main",
        pull_request: null,
        commit_sha: "7fcca06c1b2b2c482df382248610d46cfd789837",
        commit_message: "refactor(ui): display only last processed event in event source node (#315)",
        branch: {
          name: "main",
          commit_range: "2364960799e343f8cb594a81b1f34e7219f8254a...7fcca06c1b2b2c482df382248610d46cfd789837"
        }
      },
      workflow: {
        initial_pipeline_id: crypto.randomUUID(),
        id: crypto.randomUUID(),
        created_at: new Date().toISOString()
      },
      pipeline: {
        yaml_file_name: "semaphore.yml",
        working_directory: ".semaphore",
        stopping_at: new Date().toISOString(),
        state: "done",
        running_at: new Date().toISOString(),
        result_reason: "",
        result: "passed",
        queuing_at: new Date().toISOString(),
        pending_at: new Date().toISOString(),
        name: "Pipeline",
        id: crypto.randomUUID(),
        error_description: "",
        done_at: new Date().toISOString(),
        created_at: new Date().toISOString()
      },
      blocks: [
        {
          state: "done",
          result_reason: "",
          result: "passed",
          name: "List & Test & Build",
          jobs: [
            {
              status: "finished",
              result: "passed",
              name: "Test",
              index: 1,
              id: crypto.randomUUID()
            },
            {
              status: "finished",
              result: "passed",
              name: "Build",
              index: 2,
              id: crypto.randomUUID()
            },
            {
              status: "finished",
              result: "passed",
              name: "Lint",
              index: 0,
              id: crypto.randomUUID()
            }
          ]
        }
      ]
    })
  },
  {
    name: 'Custom WebHook',
    description: 'Create your own custom webhook event with custom data',
    icon: 'webhook',
    eventType: 'custom',
    nodeType: 'event_source' as const,
    getEventData: () => ({})
  },
  {
    name: 'SuperPlane - Execution Finished Event',
    description: 'Event emitted when a SuperPlane stage execution completes',
    icon: 'task_alt',
    eventType: 'execution_finished',
    nodeType: 'stage' as const,
    getEventData: () => ({
      type: "execution_finished",
      stage: {
        id: crypto.randomUUID()
      },
      execution: {
        created_at: new Date().toISOString(),
        finished_at: new Date().toISOString(),
        id: crypto.randomUUID(),
        result: "passed",
        result_message: "",
        result_reason: ""
      },
    })
  }
];