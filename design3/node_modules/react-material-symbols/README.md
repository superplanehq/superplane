# React Material Symbols ![NodeJS CI](https://github.com/ed-software/react-material-symbols/actions/workflows/ci.yml/badge.svg) ![Publish](https://github.com/ed-software/react-material-symbols/actions/workflows/publish.yml/badge.svg) ![Release](https://github.com/ed-software/react-material-symbols/actions/workflows/release.yml/badge.svg)

A simple package for adding [Material Symbols](https://fonts.google.com/icons?icon.set=Material+Symbols) to any React project.

![foo](/preview.png "icons preview")

## Authors

- [@ed-software](https://www.github.com/ed-software)

## Installation

Install with npm

```bash
  npm i react-material-symbols
```

## Props

#### `type MaterialSymbolProps`

| Prop     | Type                                                     | Description                                                                                                                                                                                |
| :------- | :------------------------------------------------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `icon`   | `'10k' \| '10mp' \| ... 2368 more ... \| 'zoom_out_map'` | **Required**. The name of the icon to render.                                                                                                                                              |
| `fill`   | `boolean \| undefined`                                   | **Default `false`**. Fill gives you the ability to modify the default icon style. A single icon can render both unfilled and filled states.                                                |
| `weight` | `100 \| 200 \| ... 6 more ... \| 900 \| undefined`       | **Default `inherit`**. Weight defines the symbol’s stroke weight, with a range of weights between thin (100) and heavy (900). Weight can also affect the overall size of the symbol.       |
| `grade`  | `number \| undefined`                                    | **Default `inherit`**. Weight and grade affect a symbol’s thickness. Adjustments to grade are more granular than adjustments to weight and have a small impact on the size of the symbol.  |
| `size`   | `number \| undefined`                                    | **Default `inherit`**. Size defines the icon width and height in pixels. For the image to look the same at different sizes, the stroke weight (thickness) changes as the icon size scales. |
| `color`  | `string \| undefined`                                    | **Default `inherit`**. Color accepts key values (`'red'`, `'blue'`, `'indigo'`, etc.), `<hex-color>`, `<rgb()>`, `<hsl()>` and `<hwb()>` values.                                           |
| `as`     | `React.ElementType \| undefined`                         | **Default `'span'`**. Set the HTML Element for the icon.                                                                                                                                   |

## Examples

```TSX
import { MaterialSymbol } from 'react-material-symbols';
import 'react-material-symbols/rounded'; // Place in your root app file. There are also `sharp` and `outlined` variants.

export default function App() {
  return (
    <MaterialSymbol icon="folder" size={24} fill grade={-25} color='red' />
  )
}

```

## Demo

<https://react-material-symbols.vercel.app/>

## License

Material Symbols are created by [Google](https://github.com/google/material-design-icons#license).

> We have made these icons available for you to incorporate into your products under the [Apache License Version 2.0](https://www.apache.org/licenses/LICENSE-2.0.txt). Feel free to remix and re-share these icons and documentation in your products. We'd love attribution in your app's about screen, but it's not required.
