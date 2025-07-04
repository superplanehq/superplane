<p align="center">
  <img width="42" height="42" alt="Motion logo" src="https://github.com/user-attachments/assets/00d6d1c3-72c4-4c2f-a664-69da13182ffc" />
</p>
<h1 align="center">Motion for React</h1>
<h3 align="center">
  An open source, production-ready animation library
</h3>
<p align="center">Previously Framer Motion. Also available for <a href="https://github.com/motiondivision/motion/tree/main/packages/motion">JavaScript</a> and <a href="https://github.com/motiondivision/motion-vue">Vue</a></p>

<br>

<p align="center">
  <a href="https://www.npmjs.com/package/framer-motion" target="_blank">
    <img src="https://img.shields.io/npm/v/framer-motion.svg?style=flat-square" />
  </a>
  <a href="https://www.npmjs.com/package/framer-motion" target="_blank">
  <img src="https://img.shields.io/npm/dm/framer-motion.svg?style=flat-square" />
  </a>
  <a href="https://twitter.com/mattgperry" target="_blank">
  <img src="https://img.shields.io/twitter/follow/mattgperry.svg?style=social&label=Follow"  />
  </a>
</p>

<br>

Motion for React is an open source, production-ready library that’s designed for all creative developers.

It's the only animation library with a hybrid engine, combining the power of JavaScript animations with the performance of native browser APIs.

It looks like this:

```jsx
<motion.div animate={{ x: 0 }} />
```

It does all this:

-   [Springs](https://motion.dev/docs/react-transitions#spring)
-   [Keyframes](https://motion.dev/docs/react-animation#keyframes)
-   [Layout animations](https://motion.dev/docs/react-layout-animations)
-   [Shared layout animations](https://motion.dev/docs/react-layout-animations#shared-layout-animations)
-   [Gestures (drag/tap/hover)](https://motion.dev/docs/react-gestures)
-   [Scroll animations](https://motion.dev/docs/react-scroll-animations)
-   [SVG paths](https://motion.dev/docs/react-animation#svg-line-drawing)
-   [Exit animations](https://motion.dev/docs/react-animation#exit-animations)
-   [Server-side rendering](https://motion.dev/docs/react-motion-component#server-side-rendering)
-   [Independent transforms](https://motion.dev/docs/react-motion-component#style)
-   [Orchestrate animations across components](https://motion.dev/docs/react-animation#orchestration)
-   [CSS variables](https://motion.dev/docs/react-animation#css-variables)

...and a whole lot more.

## Get started

### 🐇 Quick start

```bash
npm install motion
```

```jsx
import { motion } from "motion/react"

function Component() {
    return <motion.div animate={{ x: 100 }} />
}
```

Get started with [Motion for React](https://motion.dev/docs/react-quick-start).

## 🎨 Studio

![Video of bezier curve editing](https://framerusercontent.com/images/KO5dnHOUSNGb9S73p1J7nLhoFI.gif)

Motion Studio is a versatile suite of developer tools allowing you to:

-   Visually edit CSS and Motion easing curves in VS Code
-   Generate CSS springs with LLMs
-   Load Motion docs into your LLM

Get started with [Motion Studio](https://motion.dev/docs/tools-quick-start).

## 🎓 Examples

[Motion Examples](https://examples.motion.dev/react) offers 100s of free and Motion+ examples for beginners and advanced users alike. Easy copy/paste code to kickstart your next project.

## ⚡️ Motion+

[Motion+](https://motion.dev/plus) is a one-time fee, lifetime membership that unlocks over 100 premium examples, early access, powerful Studio tools, a private Discord, and exclusive APIs like:

-   Cursor
-   Ticker
-   AnimateNumber
-   splitText

[Get Motion+](https://motion.dev/plus)

### 💎 Contribute

-   Want to contribute to Motion? Our [contributing guide](https://github.com/motiondivision/motion/blob/master/CONTRIBUTING.md) has you covered.

### 👩🏻‍⚖️ License

-   Motion for React is MIT licensed.

## ✨ Sponsors

Motion is sustainable thanks to the kind support of its sponsors.

### Partners

#### Framer

Motion powers Framer animations, the web builder for creative pros. Design and ship your dream site. Zero code, maximum speed.

<a href="https://www.framer.com?utm_source=motion-readme">
  <img alt="Framer" src="https://github.com/user-attachments/assets/0404c7a1-c29d-4785-89ae-aae315f3c759" width="300px" height="200px">
</a>

### Platinum

<a href="https://tailwindcss.com"><img alt="Tailwind" src="https://github.com/user-attachments/assets/c0496f09-b8ee-4bc4-85ab-83a071bbbdec" width="300px" height="200px"></a> <a href="https://emilkowal.ski"><img alt="Emil Kowalski" src="https://github.com/user-attachments/assets/29f56b1a-37fb-4695-a6a6-151f6c24864f" width="300px" height="200px"></a> <a href="https://linear.app"><img alt="Linear" src="https://github.com/user-attachments/assets/a93710bb-d8ed-40e3-b0fb-1c5b3e2b16bb" width="300px" height="200px"></a>

### Gold

<a href="https://vercel.com"><img alt="Vercel" src="https://github.com/user-attachments/assets/23cb1e37-fa67-49ad-8f77-7f4b8eaba325" width="225px" height="150px"></a> <a href="https://liveblocks.io"><img alt="Liveblocks" src="https://github.com/user-attachments/assets/31436a47-951e-4eab-9a68-bdd54ccf9444" width="225px" height="150px"></a> <a href="https://lu.ma"><img alt="Luma" src="https://github.com/user-attachments/assets/4fae0c9d-de0f-4042-9cd6-e07885d028a9" width="225px" height="150px"></a>

### Silver

<a href="https://www.frontend.fyi/?utm_source=motion"><img alt="Frontend.fyi" src="https://github.com/user-attachments/assets/07d23aa5-69db-44a0-849d-90177e6fc817" width="150px" height="100px"></a> <a href="https://firecrawl.dev"><img alt="Firecrawl" src="https://github.com/user-attachments/assets/cba90e54-1329-4353-8fba-85beef4d2ee9" width="150px" height="100px"></a> <a href="https://puzzmo.com"><img alt="Puzzmo" src="https://github.com/user-attachments/assets/aa2d5586-e5e2-43b9-8446-db456e4b0758" width="150px" height="100px"></a> <a href="https://buildui.com"><img alt="Build UI" src="https://github.com/user-attachments/assets/024bfcd5-50e8-4b3d-a115-d5c6d6030d1c" width="150px" height="100px"></a>

### Personal

-   [OlegWock](https://sinja.io)
-   [Lambert Weller](https://github.com/l-mbert)
-   [Jake LeBoeuf](https://jklb.wf)
-   [Han Lee](https://github.com/hahnlee)
