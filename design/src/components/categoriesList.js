export const categoriesList = 
[
  {
    "category_name": "Source Control",
    "tools": [
      {
        "name": "GitHub",
        "logo": require("../images/icn-github.svg").default,
        "actions": [
          "Create Repository",
          "Clone Repository",
          "Push Changes",
          "Create Branch",
          "Merge Branch",
          "Create Pull Request",
          "Close Pull Request",
          "Add Collaborator",
          "Set Branch Protection Rule",
          "Delete Branch"
        ],
        "events": [
          "Push Event",
          "Pull Request Opened",
          "Pull Request Merged",
          "Issue Opened",
          "Issue Closed",
          "Comment Created",
          "Branch Created",
          "Branch Deleted",
          "Repository Created",
          "Fork Created"
        ]
      },
      {
        "name": "GitLab",
        "logo": require("../images/icn-gitlab.svg").default,
        "actions": [
          
        ],
        "events": [
         
        ]
      },
      {
        "name": "Bitbucket",
        "logo": require("../images/icn-bitbucket.svg").default,
        "actions": [
          "Create Repository",
          "Clone Repository",
          "Push Commits",
          "Create Branch",
          "Merge Pull Request",
          "Open Pull Request",
          "Decline Pull Request",
          "Add User to Team",
          "Configure Branch Permissions",
          "Delete Repository"
        ],
        "events": [
          "Repo Push",
          "Pull Request Created",
          "Pull Request Merged",
          "Issue Created",
          "Issue Updated",
          "Comment Added",
          "Branch Created",
          "Branch Deleted",
          "Repository Forked",
          "Repository Pushed"
        ]
      },
      {
        "name": "Azure",
        "logo": require("../images/logos/azure.svg").default,
        "actions": [
          "Create Azure DevOps Project",
          "Clone Git Repository",
          "Push to Repository",
          "Create New Branch",
          "Complete Pull Request",
          "Create Pull Request",
          "Abandon Pull Request",
          "Add Team Member",
          "Set Repository Policy",
          "Delete Repository"
        ],
        "events": [
          "Code Pushed",
          "Pull Request Created",
          "Pull Request Completed",
          "Work Item Created",
          "Work Item Updated",
          "Comment Added",
          "Branch Created",
          "Branch Deleted",
          "Repository Created",
          "Build Completed"
        ]
      }
    ]
  },
  {
    "category_name": "CI/CD Pipeline",
    "tools": [
      {
        "name": "Jenkins",
        "logo": require("../images/logos/jenkins.svg").default,
        "actions": [
          "Create Freestyle Project",
          "Configure Pipeline Job",
          "Trigger Build",
          "Abort Build",
          "Deploy Artifact",
          "Run Tests",
          "Archive Artifacts",
          "Add Build Step",
          "Set Up Webhook",
          "Delete Job"
        ],
        "events": [
          "Build Started",
          "Build Succeeded",
          "Build Failed",
          "Build Aborted",
          "Deployment Successful",
          "Deployment Failed",
          "Job Configured",
          "Agent Connected",
          "Pipeline Stage Completed",
          "Test Results Published"
        ]
      },
      {
        "name": "GitHub",
        "logo": require("../images/icn-github.svg").default,
        "actions": [
          "Create Workflow File",
          "Define Job",
          "Run Workflow Manually",
          "Cancel Workflow Run",
          "Deploy to Environment",
          "Publish Package",
          "Set Up Secrets",
          "Add Workflow Dispatch Trigger",
          "Configure Environment Protection Rule",
          "Disable Workflow"
        ],
        "events": [
          "Workflow Run Started",
          "Workflow Run Succeeded",
          "Workflow Run Failed",
          "Workflow Run Canceled",
          "Deployment Succeeded",
          "Deployment Failed",
          "Pull Request Sync",
          "Push to Branch",
          "Release Published",
          "Scheduled Event"
        ]
      },
      {
        "name": "GitLab",
        "logo": require("../images/icn-gitlab.svg").default,
        "actions": [
          "Create .gitlab-ci.yml",
          "Define CI/CD Job",
          "Run Pipeline Manually",
          "Cancel Running Pipeline",
          "Deploy to Production",
          "Build Docker Image",
          "Set CI/CD Variable",
          "Add Schedule Trigger",
          "Configure Environment Scope",
          "Pause Pipeline"
        ],
        "events": [
          "Pipeline Started",
          "Pipeline Succeeded",
          "Pipeline Failed",
          "Pipeline Canceled",
          "Job Succeeded",
          "Job Failed",
          "Merge Request Pipeline",
          "Push Pipeline",
          "Tag Pipeline",
          "Scheduled Pipeline"
        ]
      },
      {
        "name": "CircleCI",
        "logo": require("../images/logos/circleci.svg").default,
        "actions": [
          "Create .circleci/config.yml",
          "Define Workflow",
          "Trigger Build",
          "Cancel Build",
          "Deploy Application",
          "Run Tests",
          "Set Environment Variable",
          "Add Orb",
          "Configure Context",
          "Rerun Workflow"
        ],
        "events": [
          "Build Started",
          "Build Succeeded",
          "Build Failed",
          "Build Canceled",
          "Deployment Completed",
          "Workflow Completed",
          "Scheduled Workflow",
          "API Triggered Build",
          "Commit Pushed",
          "Pull Request Opened"
        ]
      },
      {
        "name": "Azure",
        "logo": require("../images/logos/azure.svg").default,
        "actions": [
          "Create Azure Pipeline",
          "Define Build Stage",
          "Run Pipeline",
          "Cancel Pipeline Run",
          "Deploy to Azure App Service",
          "Publish Test Results",
          "Add Pipeline Variable Group",
          "Set Up Release Gate",
          "Configure Agent Pool",
          "Disable Pipeline"
        ],
        "events": [
          "Pipeline Run Started",
          "Pipeline Run Succeeded",
          "Pipeline Run Failed",
          "Pipeline Run Canceled",
          "Deployment Completed",
          "Build Completed",
          "Code Pushed",
          "Pull Request Updated",
          "Scheduled Build",
          "Release Created"
        ]
      },
      {
        "name": "TeamCity",
        "logo": require("../images/logos/teamcity-icon.svg").default,
        "actions": [
          "Create Build Configuration",
          "Add VCS Root",
          "Trigger Build",
          "Stop Build",
          "Deploy to Server",
          "Run Inspections",
          "Configure Build Step",
          "Set Up Build Trigger",
          "Add Agent Requirement",
          "Archive Build"
        ],
        "events": [
          "Build Started",
          "Build Finished Successfully",
          "Build Failed",
          "Build Interrupted",
          "Deployment Completed",
          "VCS Change Detected",
          "Agent Status Changed",
          "Project Created",
          "Build Parameter Changed",
          "Test Failed"
        ]
      },
      {
        "name": "Bamboo",
        "logo": require("../images/logos/bamboo-logo.svg").default,
        "actions": [
          "Create Deployment Project",
          "Configure Plan",
          "Run Plan",
          "Stop Plan Execution",
          "Deploy Release",
          "Execute Tests",
          "Add Task to Job",
          "Set Up Trigger",
          "Configure Agent Capability",
          "Disable Plan"
        ],
        "events": [
          "Plan Started",
          "Plan Completed Successfully",
          "Plan Failed",
          "Plan Stopped",
          "Deployment Started",
          "Deployment Succeeded",
          "Repository Polled",
          "Build Queued",
          "Artifact Produced",
          "Test Result Updated"
        ]
      }
    ]
  },
  {
    "category_name": "Container & Orchestration",
    "tools": [
      {
        "name": "Docker",
        "logo": require("../images/logos/lang-docker.svg").default,
        "actions": [
          "Build Image",
          "Run Container",
          "Push Image to Registry",
          "Pull Image from Registry",
          "Create Network",
          "Create Volume",
          "Stop Container",
          "Remove Container",
          "Inspect Image",
          "Login to Registry"
        ],
        "events": [
          "Container Started",
          "Container Stopped",
          "Image Built",
          "Image Pushed",
          "Volume Created",
          "Network Created",
          "Container Died",
          "Image Pulled",
          "Container OOMKilled",
          "Daemon Reload"
        ]
      },
      {
        "name": "Kubernetes",
        "logo": require("../images/logos/kubernetes.svg").default,
        "actions": [
          "Deploy Pod",
          "Create Service",
          "Scale Deployment",
          "Update Deployment",
          "Delete Pod",
          "Apply Manifest",
          "Get Pod Logs",
          "Execute Command in Pod",
          "Create Namespace",
          "Configure Ingress"
        ],
        "events": [
          "Pod Created",
          "Pod Deleted",
          "Deployment Updated",
          "Service Created",
          "Node Ready",
          "Container Started",
          "Container Terminated",
          "ReplicaSet Scaled",
          "Endpoint Added",
          "ConfigMap Updated"
        ]
      },
      {
        "name": "Docker",
        "logo": require("../images/logos/lang-docker.svg").default,
        "actions": [
          "Bring Up Stack",
          "Bring Down Stack",
          "Build Services",
          "Scale Service",
          "View Logs",
          "Restart Service",
          "Create Compose File",
          "Validate Compose File",
          "List Services",
          "Execute Command in Service"
        ],
        "events": [
          "Service Started",
          "Service Stopped",
          "Stack Up",
          "Stack Down",
          "Service Recreated",
          "Container Created",
          "Container Removed",
          "Network Created",
          "Volume Created",
          "Lint Chart",
          "Package Chart",
          "Add Repository",
          "Update Repositories",
          "List Releases",
          "Get Release Status"
        ],
        "events": [
          "Release Installed",
          "Release Upgraded",
          "Release Rolled Back",
          "Release Uninstalled",
          "Chart Packaged",
          "Repository Added",
          "Repository Updated",
          "Chart Downloaded",
          "Hook Executed",
          "Pre-install Hook Failed"
        ]
      },
      {
        "name": "OpenShift",
        "logo": require("../images/logos/openshift.svg").default,
        "actions": [
          "Create Application",
          "Deploy Image",
          "Scale DeploymentConfig",
          "Create Route",
          "Delete Pod",
          "Apply Template",
          "Get Build Logs",
          "Start Build",
          "Create Project",
          "Add Role Binding"
        ],
        "events": [
          "Pod Created",
          "DeploymentConfig Scaled",
          "Route Created",
          "Build Started",
          "Build Succeeded",
          "Build Failed",
          "ImageStream Tagged",
          "Project Created",
          "Service Created",
          "Container Running"
        ]
      },
      {
        "name": "Amazon",
        "logo": require("../images/logos/acm.svg.svg").default,
        "actions": [
          "Run ECS Task",
          "Create ECS Service",
          "Update ECS Service",
          "Register Task Definition",
          "Deregister Task Definition",
          "Start EC2 Instance",
          "Stop EC2 Instance",
          "Create Cluster",
          "Delete Cluster",
          "Scale Service"
        ],
        "events": [
          "Task Started",
          "Task Stopped",
          "Service Deployed",
          "Service Updated",
          "Instance Launched",
          "Instance Terminated",
          "Cluster Created",
          "Task Definition Registered",
          "Container Instance Registered",
          "Deployment Completed"
        ]
      }
    ]
  },
  {
    "category_name": "Cloud Infrastructure",
    "tools": [
      {
        "name": "AWS",
        "logo": require("../images/logos/aws-cloudformation.svg").default,
        "actions": [
          "Launch EC2 Instance",
          "Stop EC2 Instance",
          "Terminate EC2 Instance",
          "Create S3 Bucket",
          "Upload Object to S3",
          "Create Lambda Function",
          "Invoke Lambda Function",
          "Create VPC",
          "Configure Security Group",
          "Attach EBS Volume"
        ],
        "events": [
          "Instance State Change",
          "S3 Object Created",
          "S3 Object Deleted",
          "Lambda Invoked",
          "VPC Created",
          "Security Group Modified",
          "EBS Volume Attached",
          "CloudWatch Alarm State Change",
          "Auto Scaling Group Launch",
          "API Call Made"
        ]
      },
      {
        "name": "Azure",
        "logo": require("../images/logos/azure.svg").default,
        "actions": [
          "Create Resource Group",
          "Deploy ARM Template",
          "Create Virtual Machine",
          "Start Virtual Machine",
          "Stop Virtual Machine",
          "Delete Resource Group",
          "Create Storage Account",
          "Upload Blob",
          "Create Function App",
          "Deploy Function"
        ],
        "events": [
          "Resource Group Created",
          "Deployment Succeeded",
          "VM Started",
          "VM Stopped",
          "Resource Deleted",
          "Blob Uploaded",
          "Function Executed",
          "Resource Updated",
          "Subscription Quota Exceeded",
          "Activity Log Alert"
        ]
      },
     
      {
        "name": "CloudFormation",
        "logo": require("../images/logos/aws-cloudformation.svg").default,
        "actions": [
          "Create Stack",
          "Update Stack",
          "Delete Stack",
          "Validate Template",
          "Estimate Cost",
          "Drift Detection",
          "Create Change Set",
          "Execute Change Set",
          "Describe Stack Resources",
          "Cancel Update Stack"
        ],
        "events": [
          "Stack Create Complete",
          "Stack Update Complete",
          "Stack Delete Complete",
          "Resource Create Complete",
          "Resource Update Complete",
          "Resource Delete Complete",
          "Stack Rollback Complete",
          "Change Set Created",
          "Change Set Executed",
          "Stack Drifted"
        ]
      },
      {
        "name": "Pulumi",
        "logo": require("../images/logos/pulumi.svg").default,
        "actions": [
          "Create Stack",
          "Select Stack",
          "Up (Deploy)",
          "Destroy (Delete)",
          "Refresh State",
          "Preview Changes",
          "Export Stack",
          "Import Resource",
          "Configure Stack Setting",
          "Login to Backend"
        ],
        "events": [
          "Stack Created",
          "Deployment Started",
          "Deployment Succeeded",
          "Deployment Failed",
          "Resource Created",
          "Resource Updated",
          "Resource Deleted",
          "Stack Destroyed",
          "State Refreshed",
          "Preview Generated"
        ]
      }
    ]
  },
  {
    "category_name": "Monitoring & Observability",
    "tools": [
      {
        "name": "Datadog",
        "logo": require("../images/logos/datadog.svg").default,
        "actions": [
          "Create Monitor",
          "Mute Monitor",
          "Unmute Monitor",
          "Create Dashboard",
          "Add Widget to Dashboard",
          "Send Event",
          "Install Agent",
          "Configure Integration",
          "Query Metrics",
          "Create Log Pipeline"
        ],
        "events": [
          "Alert Triggered",
          "Monitor Resolved",
          "Monitor Warned",
          "Event Received",
          "Metric Submitted",
          "Log Received",
          "Integration Configured",
          "Dashboard Created",
          "Agent Status Change",
          "Monitor Muted"
        ]
      },
      {
        "name": "Grafana",
        "logo": require("../images/logos/grafana.svg").default,
        "actions": [
          "Create Dashboard",
          "Add Panel to Dashboard",
          "Configure Data Source",
          "Create Alert Rule",
          "Send Notification",
          "Import Dashboard",
          "Create User",
          "Set Permissions",
          "Annotate Dashboard",
          "Share Dashboard"
        ],
        "events": [
          "Alert Triggered",
          "Alert Resolved",
          "Dashboard Created",
          "Panel Added",
          "Data Source Connected",
          "Annotation Created",
          "User Logged In",
          "Dashboard Viewed",
          "Notification Sent",
          "Panel Data Refreshed"
        ]
      },
      {
        "name": "PagerDuty",
        "logo": require("../images/logos/pagerduty.svg").default,
        "actions": [
          "Create Service",
          "Create Escalation Policy",
          "Trigger Incident",
          "Acknowledge Incident",
          "Resolve Incident",
          "Add User",
          "Create On-Call Schedule",
          "Send Custom Event",
          "Add Integration",
          "Set Maintenance Window"
        ],
        "events": [
          "Incident Triggered",
          "Incident Acknowledged",
          "Incident Resolved",
          "Incident Escalated",
          "Service Created",
          "User Added",
          "Schedule Change",
          "Alert Received",
          "Integration Activated",
          "Maintenance Window Started"
        ]
      },
    ]
  },
  {
    "category_name": "Testing & Quality",
    "tools": [
      {
        "name": "SonarQube",
        "logo": require("../images/logos/sonarqube-1.svg").default,
        "actions": [
          "Run Code Scan",
          "Analyze Project",
          "Create Quality Gate",
          "Set Quality Profile",
          "Exclude Files from Analysis",
          "Add User to Group",
          "Configure Webhook",
          "Mark Issue as False Positive",
          "Create Project",
          "Delete Project"
        ],
        "events": [
          "Analysis Completed",
          "Quality Gate Passed",
          "Quality Gate Failed",
          "New Code Smells Detected",
          "New Bugs Detected",
          "New Vulnerabilities Detected",
          "Project Created",
          "Webhook Triggered",
          "Issue Commented",
          "Issue Resolved"
        ]
      },
      {
        "name": "Selenium",
        "logo": require("../images/logos/selenium.svg").default,
        "actions": [
          "Open URL",
          "Click Element",
          "Type Text",
          "Submit Form",
          "Take Screenshot",
          "Find Element by ID",
          "Find Element by XPath",
          "Wait for Element",
          "Execute JavaScript",
          "Close Browser"
        ],
        "events": [
          "Page Loaded",
          "Element Clicked",
          "Text Entered",
          "Form Submitted",
          "Screenshot Taken",
          "Test Case Passed",
          "Test Case Failed",
          "Browser Opened",
          "Browser Closed",
          "Element Found"
        ]
      },
      {
        "name": "Jest",
        "logo": require("../images/logos/jest.svg").default,
        "actions": [
          "Run Tests",
          "Watch Tests",
          "Generate Coverage Report",
          "Configure Test Environment",
          "Mock Module",
          "Snapshot Test",
          "Run Specific Test File",
          "Clear Mocks",
          "Set Timeout",
          "Use Global Setup/Teardown"
        ],
        "events": [
          "Test Suite Started",
          "Test Suite Passed",
          "Test Suite Failed",
          "Test Case Passed",
          "Test Case Failed",
          "Coverage Report Generated",
          "Snapshot Updated",
          "Hook Executed",
          "Test File Watched",
          "Error Thrown in Test"
        ]
      },
      {
        "name": "Postman",
        "logo": require("../images/logos/postman.svg").default,
        "actions": [
          "Send API Request",
          "Create Collection",
          "Add Request to Collection",
          "Run Collection (Newman)",
          "Set Environment Variable",
          "Write Test Script",
          "Import OpenAPI Spec",
          "Export Collection",
          "Share Collection",
          "Generate Code Snippet"
        ],
        "events": [
          "Request Sent",
          "Response Received",
          "Test Script Executed",
          "Collection Run Started",
          "Collection Run Completed",
          "Environment Variable Set",
          "Request Failed",
          "Assertion Failed",
          "Collection Imported",
          "Collection Shared"
        ]
      }
    ]
  }
]
