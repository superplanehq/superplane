export const toolsList = [
  {
    name: "Github",
    logo: require("../images/github.svg").default,
    actions: [
          "Create a file",
          "Delete a file",
          "Edit a file",
          "Get a file",
          "List files",
          "Create an issue",
          "Create a comment on an issue",
          "Edit an issue",
          "Get an issue",
          "Lock an issue",
          "Get repositories for an organization",
          "Create a release",
          "Delete a release",
          "Get a release",
          "Get many releases",
          "Update a release",
          "Get a repository",
          "Get issues of a repository",
          "Get the license of a repository",
          "Get the profile of a repository",
          "Get pull requests of a repository",
          "List popular paths in a repository"
      ],
    events: [
      "On check run",
        "On check suite",
        "On commit comment",
        "On create",
        "On delete",
        "On deploy key",
        "On deployment",
        "On deployment status",
        "On fork",
        "On github app authorization",
        "On gollum",
        "On installation",
        "On installation repositories",
        "On issue comment",
        "On issues",
        "On label"
        ]
    
  },
  {
    name: "Ruby",
    logo: require("../images/logos/lang-ruby.svg").default,
    actions: [
      "Create Ruby script",
      "Run Ruby script",
      "Install Ruby gem",
      "Update Ruby gem",
      "List installed gems",
      "Run RSpec tests",
      "Run RubyMine analysis"
    ],
    events: [
      "On Ruby script execution",
      "On gem installation",
      "On gem update",
      "On test completion"
    ]
  },
  {
    name: "PHP",
    logo: require("../images/logos/lang-php.svg").default,
    actions: [
      "Create PHP script",
      "Run PHP script",
      "Install Composer package",
      "Update Composer package",
      "Run PHPUnit tests",
      "Run PHPStan analysis",
      "Run PHP CS Fixer"
    ],
    events: [
      "On PHP script execution",
      "On Composer package installation",
      "On test completion",
      "On code style check"
    ]
  },
  {
    name: "Django",
    logo: require("../images/logos/lang-django.svg").default,
    actions: [
      "Create Django app",
      "Run migrations",
      "Create superuser",
      "Run tests",
      "Collect static files",
      "Run development server",
      "Create model"
    ],
    events: [
      "On migration",
      "On server start",
      "On test completion",
      "On model creation"
    ]
  },
  {
    name: "Python",
    logo: require("../images/logos/lang-python.svg").default,
    actions: [
      "Create Python script",
      "Run Python script",
      "Install pip package",
      "Update pip package",
      "Run pytest tests",
      "Run black formatting",
      "Run mypy type checking"
    ],
    events: [
      "On Python script execution",
      "On package installation",
      "On test completion",
      "On code formatting"
    ]
  },
  {
    name: "Yarn",
    logo: require("../images/logos/lang-yarn.svg").default,
    actions: [
      "Install dependencies",
      "Add package",
      "Remove package",
      "Run script",
      "List dependencies",
      "Update dependencies",
      "Create workspace"
    ],
    events: [
      "On dependency installation",
      "On package addition",
      "On script execution",
      "On workspace creation"
    ]
  },
  {
    name: "Scala",
    logo: require("../images/logos/lang-scala.svg").default,
    actions: [
      "Create Scala project",
      "Compile Scala code",
      "Run Scala application",
      "Run Scala tests",
      "Run sbt task",
      "Create Scala class",
      "Create Scala trait"
    ],
    events: [
      "On compilation",
      "On test completion",
      "On task execution",
      "On class creation"
    ]
  },
  {
    name: "Rails",
    logo: require("../images/logos/lang-rails.svg").default,
    actions: [
      "Create Rails app",
      "Run migrations",
      "Create controller",
      "Create model",
      "Create view",
      "Run tests",
      "Run development server"
    ],
    events: [
      "On migration",
      "On server start",
      "On test completion",
      "On controller creation"
    ]
  },
  {
    name: "Laravel",
    logo: require("../images/logos/lang-laravel.svg").default,
    actions: [
      "Create Laravel app",
      "Run migrations",
      "Create controller",
      "Create model",
      "Create view",
      "Run tests",
      "Run artisan command"
    ],
    events: [
      "On migration",
      "On command execution",
      "On test completion",
      "On controller creation"
    ]
  },
  {
    name: "Fastlane",
    logo: require("../images/logos/lang-fastlane.svg").default,
    actions: [
      "Run lane",
      "Deploy to App Store",
      "Deploy to Google Play",
      "Create screenshots",
      "Build app",
      "Test app",
      "Create release notes"
    ],
    events: [
      "On lane execution",
      "On deployment",
      "On build completion",
      "On test completion"
    ]
  },
  {
    name: "Phoenix",
    logo: require("../images/logos/lang-phoenix.svg").default,
    actions: [
      "Create Phoenix app",
      "Run migrations",
      "Create controller",
      "Create context",
      "Create view",
      "Run tests",
      "Run development server"
    ],
    events: [
      "On migration",
      "On server start",
      "On test completion",
      "On controller creation"
    ]
  },
  {
    name: "Docker",
    logo: require("../images/logos/lang-docker.svg").default,
    actions: [
      "Build image",
      "Run container",
      "Push image",
      "Pull image",
      "Create network",
      "Create volume",
      "Run command in container"
    ],
    events: [
      "On image build",
      "On container start",
      "On image push",
      "On command execution"
    ]
  },
  {
    name: "textract",
    logo: require('../images/logos/textract.svg fill.svg').default,
    actions: [
      "Extract text from PDF",
      "Extract text from image",
      "Extract text from Word",
      "Extract text from Excel",
      "Extract text from PowerPoint",
      "Extract metadata",
      "Process document"
    ],
    events: [
      "On text extraction",
      "On metadata extraction",
      "On document processing",
      "On format conversion"
    ]
  },
  {
    name: "sqs",
    logo: require('../images/logos/sqs.svg fill.svg').default,
    actions: [
      "Create queue",
      "Send message",
      "Receive message",
      "Delete message",
      "List queues",
      "Set queue attributes",
      "Receive messages in batch"
    ],
    events: [
      "On message sent",
      "On message received",
      "On queue creation",
      "On attribute update"
    ]
  }
]
/*export const toolsList2 = [
    {
      "name": "Ruby",
      "logo": require("../images/logos/lang-ruby.svg").default
    },
    {
      "name": "PHP",
      "logo": require("../images/logos/lang-php.svg").default
    },
    {
      "name": "Django",
      "logo": require("../images/logos/lang-django.svg").default
    },
    {
      "name": "Python",
      "logo": require("../images/logos/lang-python.svg").default
    },
    {
      "name": "Yarn",
      "logo": require("../images/logos/lang-yarn.svg").default
    },
    {
      "name": "Scala",
      "logo": require("../images/logos/lang-scala.svg").default
    },
    {
      "name": "Rails",
      "logo": require("../images/logos/lang-rails.svg").default
    },
    {
      "name": "Laravel",
      "logo": require("../images/logos/lang-laravel.svg").default
    },
    {
      "name": "Fastlane",
      "logo": require("../images/logos/lang-fastlane.svg").default
    },
    {
      "name": "Phoenix",
      "logo": require("../images/logos/lang-phoenix.svg").default
    },
    {
      "name": "Docker",
      "logo": require("../images/logos/lang-docker.svg").default
    },
    {
      "name": "textract",
      "logo": require('../images/logos/textract.svg fill.svg').default
    },
    {
      "name": "sqs",
      "logo": require('../images/logos/sqs.svg fill.svg').default
    },
    {
      "name": "sns",
      "logo": require('../images/logos/sns.svg fill.svg').default
    },
    {
      "name": "ses",
      "logo": require('../images/logos/ses.svg fill.svg').default
    },
    {
      "name": "s3",
      "logo": require('../images/logos/s3.svg.svg').default
    },
    {
      "name": "rekognition",
      "logo": require('../images/logos/rekognition.svg fill.svg').default
    },
    {
      "name": "lambda",
      "logo": require('../images/logos/lambda.svg fill.svg').default
    },
    {
      "name": "elb",
      "logo": require('../images/logos/elb.svg.svg').default
    },
    {
      "name": "dynamodb",
      "logo": require('../images/logos/dynamodb.svg.svg').default
    },
    {
      "name": "comprehend",
      "logo": require('../images/logos/comprehend.svg.svg').default
    },
    {
      "name": "cognito",
      "logo": require('../images/logos/cognito.svg.svg').default
    },
    {
      "name": "acm",
      "logo": require('../images/logos/acm.svg.svg').default
    },
    {
      "name": "autopilot",
      "logo": require('../images/logos/autopilot.svg.svg').default
    },
    {
      "name": "asana",
      "logo": require('../images/logos/asana.svg.svg').default
    },
    {
      "name": "apiTemplateIo",
      "logo": require('../images/logos/apiTemplateIo.svg.svg').default
    },
    {
      "name": "amqp",
      "logo": require('../images/logos/amqp.svg.svg').default
    },
    {
      "name": "alienVault",
      "logo": require('../images/logos/AlienVault.png.svg').default
    },
    {
      "name": "airtop",
      "logo": require('../images/logos/airtop.svg.svg').default
    },
    {
      "name": "airtable",
      "logo": require('../images/logos/airtable.svg.svg').default
    },
    {
      "name": "agilecrm",
      "logo": require('../images/logos/agilecrm.png.svg').default
    },
    {
      "name": "affinity",
      "logo": require('../images/logos/affinity.dark.svg.svg').default
    },
    {
      "name": "adalo",
      "logo": require('../images/logos/adalo.svg.svg').default
    },
    {
      "name": "acuityScheduling",
      "logo": require('../images/logos/acuityScheduling.png.svg').default
    },
    {
      "name": "activeCampaign",
      "logo": require('../images/logos/activeCampaign.dark.svg.svg').default
    },
    {
      "name": "actionNetwork",
      "logo": require('../images/logos/actionNetwork.svg.svg').default
    },
    {
      "name": "Salesforce",
      "logo": require("../images/logos/salesforce.svg.svg").default
    },
    {
      "name": "S3",
      "logo": require("../images/logos/s3.svg.svg").default
    },
    {
      "name": "Rundeck",
      "logo": require("../images/logos/rundeck.png.svg").default
    },
    {
      "name": "RS",
      "logo": require("../images/logos/Img.svg").default
    },
    {
      "name": "Rocket Chat",
      "logo": require("../images/logos/rocketchat.svg.svg").default
    },
    {
    "name": "Redis",
      "logo": require("../images/logos/redis.svg.svg").default
    },
    {
      "name": "Reddit",
      "logo": require("../images/logos/reddit.svg.svg").default
    },
    {
      "name": "Recorded Future",
      "logo": require("../images/logos/Background-2.svg").default
    },
    {
      "name": "Rapid7 InsightVM",
      "logo": require("../images/logos/Rapid7InsightVm.svg.svg").default
    },
    {
      "name": "Raindrop",
      "logo": require("../images/logos/raindrop.svg.svg").default
    },
    {
      "name": "RabbitMQ",
      "logo": require("../images/logos/rabbitmq.svg.svg").default
    },
    {
      "name": "QuickChart",
      "logo": require("../images/logos/quickChart.svg.svg").default
    },
    {
      "name": "QuickBooks",
      "logo": require("../images/logos/quickbooks.svg.svg").default
    },
    {
      "name": "Quickbase",
      "logo": require("../images/logos/quickbase.png.svg").default
    },
    {
      "name": "QuestDB",
      "logo": require("../images/logos/questdb.png.svg").default
    },
    {
      "name": "Qualys",
      "logo": require("../images/logos/Qualys.svg.svg").default
    },
    {
      "name": "Pushover",
      "logo": require("../images/logos/pushover.svg.svg").default
    },
    {
      "name": "Pushcut",
      "logo": require("../images/logos/pushcut.png.svg").default
    },
    {
      "name": "Pushbullet",
      "logo": require("../images/logos/pushbullet.svg.svg").default
    },
    {
      "name": "ProfitWell",
      "logo": require("../images/logos/profitwell.dark.svg.svg").default
    },
    {
      "name": "Postmark",
      "logo": require("../images/logos/postmark.png.svg").default
    },
    {
      "name": "PostHog",
      "logo": require("../images/logos/postHog.svg.svg").default
    },
    {
      "name": "Postgres",
      "logo": require("../images/logos/postgres.svg.svg").default
    },
    {
      "name": "Postbin",
      "logo": require("../images/logos/postbin.svg.svg").default
    },
    {
      "name": "Plivo",
      "logo": require("../images/logos/plivo.svg.svg").default
    },
    {
      "name": "Pipedrive",
      "logo": require("../images/logos/pipedrive.svg.svg").default
    },
    {
      "name": "Philips Hue",
      "logo": require("../images/logos/philipshue.png.svg").default
    },
    {
      "name": "PhantomBuster",
      "logo": require("../images/logos/phantombuster.png.svg").default
    },
    {
      "name": "Peekalink",
      "logo": require("../images/logos/peekalink.png.svg").default
    },
    {
      "name": "PDF",
      "logo": require("../images/logos/PDF_co_Api_56638c854f.svg.svg").default
    },
    {
      "name": "PayPal",
      "logo": require("../images/logos/paypal.svg.svg").default
    },
    {
      "name": "PagerDuty",
      "logo": require("../images/logos/pagerDuty.svg.svg").default
    },
    {
      "name": "Paddle",
      "logo": require("../images/logos/paddle.png.svg").default
    },
    {
      "name": "Oura",
      "logo": require("../images/logos/oura.dark.svg.svg").default
    },
    {
      "name": "OpenWeatherMap",
      "logo": require("../images/logos/openWeatherMap.svg.svg").default
    },
    {
      "name": "OpenThesaurus",
      "logo": require("../images/logos/openthesaurus.png.svg").default
    },
    {
      "name": "Background",
      "logo": require("../images/logos/Background.svg").default
    },
    {
      "name": "OpenCTI",
      "logo": require("../images/logos/OpenCTI.png.svg").default
    },
    {
      "name": "Onfleet",
      "logo": require("../images/logos/Onfleet.svg.svg").default
    },
    {
      "name": "OneSimpleAPI",
      "logo": require("../images/logos/onesimpleapi.svg.svg").default
    },
    {
      "name": "Okta",
      "logo": require("../images/logos/Okta.dark.svg.svg").default
    },
    {
      "name": "Odoo",
      "logo": require("../images/logos/odoo.svg.svg").default
    },
    {
      "name": "npm",
      "logo": require("../images/logos/npm.svg.svg").default
    },
    {
      "name": "Notion",
      "logo": require("../images/logos/notion.dark.svg.svg").default
    },
    {
      "name": "NocoDB",
      "logo": require("../images/logos/nocodb.svg.svg").default
    },
    {
      "name": "Nextcloud",
      "logo": require("../images/logos/nextcloud.svg.svg").default
    },
    {
      "name": "NetScaler",
      "logo": require("../images/logos/netscaler.dark.svg.svg").default
    },
    {
      "name": "Netlify",
      "logo": require("../images/logos/netlify.svg.svg").default
    },
    {
      "name": "NASA",
      "logo": require("../images/logos/nasa.png.svg").default
    }    
  ]*/