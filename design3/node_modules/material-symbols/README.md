# [material-symbols](https://github.com/marella/material-symbols/tree/main/material-symbols)

Latest variable icon fonts and CSS for Material Symbols.

> This package is automatically updated, so it will always have the latest icons from Google.

> For fonts with fewer variations, see [`@material-symbols/font-400`](https://www.npmjs.com/package/@material-symbols/font-400)
>
> For SVGs, see [`@material-symbols/svg-400`](https://www.npmjs.com/package/@material-symbols/svg-400)

- [Installation](#installation)
- [Usage](#usage)
- [Available Icons](#available-icons)

## Installation

Install the [latest version][releases] using:

```sh
npm install material-symbols@latest
```

## Usage

Import in JS (example: `src/index.js` in Create React App, `src/main.js` in Vue CLI):

```js
import 'material-symbols';
```

or import in CSS (example: `src/styles.css` in Angular CLI):

```css
@import 'material-symbols';
```

or import in HTML:

```html
<link href="/path/to/material-symbols/index.css" rel="stylesheet">
```

To display an icon, use one of the following:

```html
<span class="material-symbols-outlined">face</span> <!-- Outlined -->
<span class="material-symbols-rounded">face</span>  <!-- Rounded -->
<span class="material-symbols-sharp">face</span>    <!-- Sharp -->
```

To customize the variable font axes (fill, weight, grade, and optical size), use:

```css
.material-symbols-outlined {
  font-variation-settings: 'FILL' 0, 'wght' 400, 'GRAD' 0, 'opsz' 48;
}
```

### Reducing Build Size

The default `index.css` includes CSS for all fonts. This may cause build tools such as webpack to copy all fonts to the build directory even if you are not using all of them. To reduce the build size, import only the styles you need. For example, if you only need outlined icons, import `outlined.css` instead of the default `index.css`:

```diff
-import 'material-symbols';
+import 'material-symbols/outlined.css';
```

<details>
<summary><strong>Show all</strong></summary><br>

Icons | CSS | Sass
:--- | :--- | :---
Outlined | outlined.css | outlined.scss
Rounded | rounded.css | rounded.scss
Sharp | sharp.css | sharp.scss

</details>

### Using Sass

Import in Sass (example: `src/styles.scss` in Angular CLI):

```scss
@import 'material-symbols';
```

If you are getting errors with webpack or Vue CLI, add this line before importing:

```scss
$material-symbols-font-path: '~material-symbols/';
```

### Using Angular `mat-icon`

To display an icon, use one of the following:

```html
<mat-icon fontSet="material-symbols-outlined">face</mat-icon>
<mat-icon fontSet="material-symbols-rounded">face</mat-icon>
<mat-icon fontSet="material-symbols-sharp">face</mat-icon>
```

## Available Icons

See [demo].

## License

Material Symbols are created by [Google](https://github.com/google/material-design-icons#license).

> We have made these icons available for you to incorporate into your products under the [Apache License Version 2.0][license]. Feel free to remix and re-share these icons and documentation in your products.
We'd love attribution in your app's *about* screen, but it's not required.

[releases]: https://github.com/marella/material-symbols/releases
[license]: https://github.com/marella/material-symbols/blob/main/material-symbols/LICENSE
[demo]: https://marella.github.io/material-symbols/demo/
