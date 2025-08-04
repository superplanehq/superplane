import { defineConfig } from '@hey-api/openapi-ts';

export default defineConfig({
  input: '../api/swagger/superplane.swagger.json',
  output: 'src/api-client',
  plugins: [{
    name: '@hey-api/client-fetch',
    throwOnError: true,
  }],
});