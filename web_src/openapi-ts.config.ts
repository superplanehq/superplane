import { defineConfig } from '@hey-api/openapi-ts';

export default defineConfig({
  input: '../api/swagger/superplane.swagger.json',
  output: 'src/api-client',
  plugins: ['@hey-api/client-fetch'],
});