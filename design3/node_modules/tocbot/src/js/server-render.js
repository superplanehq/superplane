import * as tocbot from "./index-esm.js"
import { JSDOM } from "jsdom"

export function htmlTemplate(content) {
  return `
<html>
  <body>
    <div class="js-toc-content">
      ${content}
    </div>
    <div class="js-toc">
    </div>
  </body>
</html>
`
}

export function serverRender(content) {
  const html = htmlTemplate(content)
  const { window, location } = new JSDOM(html)
  global.window = window
  global.document = window.document
  global.location = location

  // Init and get HTML content.
  tocbot.init()
  const toc = window.document.body.querySelector(".js-toc")
  return toc && toc.innerHTML
}
