superplane.component({
  label: "HTTP Request with Exponential Backoff",
  description: "Makes an HTTP GET request with exponential backoff in case of failure",
  icon: "refresh-cw",
  color: "green",

  configuration: [
    {
      name: "url",
      label: "Request URL",
      type: "url",
      required: true,
      description: "The URL to send the GET request to.",
      placeholder: "https://api.example.com/data",
    },
    {
      name: "maxRetries",
      label: "Maximum Retries",
      type: "number",
      required: true,
      description: "Maximum number of retry attempts.",
      placeholder: "5",
    },
    {
      name: "initialDelay",
      label: "Initial Delay (ms)",
      type: "number",
      required: true,
      description: "Initial delay before retrying, in milliseconds.",
      placeholder: "1000",
    },
  ],

  execute: function(ctx) {
    var config = ctx.configuration;
    var url = config.url;
    var maxRetries = config.maxRetries;
    var initialDelay = config.initialDelay;

    function exponentialBackoff(retries, delay) {
      // Make HTTP GET request
      var response;
      try {
        response = ctx.http.request("GET", url, { headers: {} });
      } catch (error) {
        ctx.log.error("Request failed: " + error.message);

        // Check if retries are left
        if (retries > 0) {
          ctx.log.info("Retrying request in " + delay + " milliseconds...");
          ctx.sleep(delay);
          return exponentialBackoff(retries - 1, delay * 2);
        } else {
          ctx.fail("error", "Request failed after " + maxRetries + " retries");
          return;
        }
      }

      // On success, emit the response
      ctx.log.info("Request succeeded!");
      ctx.emit("default", "request.success", response);
    }

    exponentialBackoff(maxRetries, initialDelay);
  },
});