# Changes to PostCSS Cascade Layers

### 1.1.1 (September 17, 2022)

- Fix pre-defined layer order in nested `@layer` rules.

### 1.1.0 (September 14, 2022)

- Add support for `@scope` and `@container` as parent rules for `@layer`

### 1.0.6 (September 7, 2022)

- Fix broken `@keyframes` in `@layer`.

### 1.0.5 (July 8, 2022)

- Fix case insensitive `@layer` matching (`@LaYeR`).
- Updated `@csstools/selector-specificity` to `2.0.2` (patch)

### 1.0.4 (June 23, 2022)

- Fix selector order with any pseudo element. This plugin will no longer re-order selectors.

### 1.0.3 (June 4, 2022)

- Update `@csstools/selector-specificity` (major)

### 1.0.2 (May 20, 2022)

- Use only simple `:not(#\#)` selectors to adjust specificity.

### 1.0.1 (May 17, 2022)

- Process CSS after most other plugins to ensure correct analysis and transformation of sugary CSS.
- Fix selector order with `:before` and other pseudo elements.

### 1.0.0 (May 12, 2022)

- Initial version
