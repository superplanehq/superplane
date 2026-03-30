// For more info, see https://github.com/storybookjs/eslint-plugin-storybook#configuration-flat-config-format
import storybook from "eslint-plugin-storybook";

import js from '@eslint/js'
import importPlugin from 'eslint-plugin-import'
import jsxA11y from 'eslint-plugin-jsx-a11y'
import react from 'eslint-plugin-react'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'


export default tseslint.config({ ignores: ['dist'] }, {
  extends: [js.configs.recommended, ...tseslint.configs.recommended],
  files: ['**/*.{ts,tsx}'],
  languageOptions: {
    ecmaVersion: 2020,
    globals: globals.browser,
  },
  plugins: {
    import: importPlugin,
    'jsx-a11y': jsxA11y,
    react,
    'react-hooks': reactHooks,
    'react-refresh': reactRefresh,
  },
  settings: {
    react: {
      version: 'detect',
    },
  },
  rules: {
    ...reactHooks.configs.recommended.rules,
    "no-unused-vars": "off",
    // Start as a warning to surface refactor targets without breaking CI.
    complexity: ["warn", { max: 15 }],
    "eqeqeq": ["warn", "always", { "null": "ignore" }],
    "max-depth": ["warn", 4],
    "max-lines": [
      "warn",
      {
        "max": 500,
        "skipBlankLines": true,
        "skipComments": true
      }
    ],
    "max-lines-per-function": [
      "warn",
      {
        "max": 120,
        "skipBlankLines": true,
        "skipComments": true
      }
    ],
    "max-params": ["warn", 5],
    "max-statements": ["warn", 25],
    "no-console": [
      "warn",
      {
        "allow": ["warn", "error"]
      }
    ],
    "no-restricted-imports": [
      "warn",
      {
        "paths": [
          {
            "name": "@storybook/react",
            "message": "Use @storybook/react-vite instead."
          }
        ]
      }
    ],
    "no-restricted-syntax": [
      "warn",
      "ForInStatement",
      "LabeledStatement",
      "WithStatement"
    ],
    "jsx-a11y/alt-text": "warn",
    "jsx-a11y/aria-role": "warn",
    "jsx-a11y/label-has-associated-control": [
      "warn",
      {
        "assert": "either"
      }
    ],
    "react/jsx-no-bind": [
      "warn",
      {
        "ignoreRefs": true,
        "allowArrowFunctions": true
      }
    ],
    "@typescript-eslint/consistent-type-imports": "warn",
    'react-refresh/only-export-components': [
      'warn',
      { allowConstantExport: true },
    ],
    "@typescript-eslint/no-unused-vars": [
      "error",
      {
        "argsIgnorePattern": "^_",
        "varsIgnorePattern": "^_",
        "ignoreRestSiblings": true
      }
    ]
  },
}, storybook.configs["flat/recommended"]);
