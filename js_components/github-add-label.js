superplane.component({
  label: "GitHub Add Label",
  description: "Add a label to a GitHub issue or pull request",
  icon: "tag",
  color: "purple",

  configuration: [
    {
      name: "owner",
      label: "Owner",
      type: "string",
      required: true,
      placeholder: "superplanehq",
    },
    {
      name: "repo",
      label: "Repository",
      type: "string",
      required: true,
      placeholder: "superplane",
    },
    {
      name: "issueNumber",
      label: "Issue Number",
      type: "expression",
      required: true,
      description: "The issue or pull request number",
      placeholder: "{{ $['trigger'].number }}",
    },
    {
      name: "label",
      label: "Label",
      type: "string",
      required: true,
      placeholder: "bug",
    },
    {
      name: "tokenSecret",
      label: "Token Secret",
      type: "secret-key",
      required: true,
      description: "Secret containing the GitHub personal access token",
    },
  ],

  setup(ctx) {
    var config = ctx.configuration;

    if (!config.owner) {
      throw new Error("owner is required");
    }

    if (!config.repo) {
      throw new Error("repo is required");
    }

    if (!config.label) {
      throw new Error("label is required");
    }
  },

  execute(ctx) {
    var config = ctx.configuration;

    var token = ctx.secrets.getKey(
      config.tokenSecret.secret,
      config.tokenSecret.key
    );

    var url =
      "https://api.github.com/repos/" +
      config.owner +
      "/" +
      config.repo +
      "/issues/" +
      config.issueNumber +
      "/labels";

    ctx.log.info(
      "Adding label '" +
        config.label +
        "' to " +
        config.owner +
        "/" +
        config.repo +
        "#" +
        config.issueNumber
    );

    var response = ctx.http.request("POST", url, {
      headers: {
        Authorization: "Bearer " + token,
        Accept: "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        labels: [config.label],
      }),
    });

    if (response.status < 200 || response.status >= 300) {
      var message =
        "GitHub API returned " +
        response.status +
        ": " +
        JSON.stringify(response.body);
      ctx.fail("error", message);
      return;
    }

    ctx.emit("default", "github.labels.added", {
      owner: config.owner,
      repo: config.repo,
      issueNumber: config.issueNumber,
      label: config.label,
      allLabels: response.body,
    });
  },
});
