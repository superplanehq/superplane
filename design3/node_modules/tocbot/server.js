import express from "express"
import next from "next"
import config from "./next.config.mjs"

const dev = process.env.NODE_ENV !== "production"
const app = next({ dev })
const handle = app.getRequestHandler()

const prefix = config.assetPrefix

app
  .prepare()
  .then(() => {
    const server = express()
    const router = express.Router()

    // use next routes
    server.use(`${prefix}`, router)
    server.use(`${prefix}/static`, express.static("static"))
    server.use(handle)

    router.get("*", (req, res) => {
      return handle(req, res)
    })

    server.listen(3001, (err) => {
      if (err) throw err
      console.log(`> Ready on http://localhost:3001${prefix}`)
    })
  })
  .catch((e) => {
    console.log(e)
  })
