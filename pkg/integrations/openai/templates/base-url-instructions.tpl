The Base URL tells SuperPlane which API endpoint to call.

Use the default value when connecting directly to OpenAI:

~~~text
{{ .DefaultBaseURL }}
~~~

Change it only when your API key belongs to an OpenAI-compatible provider.

An OpenAI-compatible provider is a service that accepts OpenAI-style API requests and returns OpenAI-style responses, but is not necessarily hosted by OpenAI. Examples include an internal OpenAI API gateway, Ollama, vLLM, or another model service that exposes OpenAI-compatible `/v1` endpoints.
