The Base URL tells SuperPlane which API endpoint to call.

Leave this empty when using OpenAI's hosted API. SuperPlane will use:

~~~text
{{ .DefaultBaseURL }}
~~~

Only set a custom Base URL when using an OpenAI-compatible provider, such as a private gateway, Ollama, vLLM, or another service that supports the OpenAI API format.

