# Frontend performance measurement

Use this checklist on a connected SuperPlane instance after you change canvas,
factory, or console loading.

## Load baseline

1. Build the UI with `make check.build.ui`.
2. Open the Network panel. Disable the cache. Reload the organization home.
3. Record JS transfer size, CSS transfer size, and time to the home page.
4. Open a canvas. Record the extra JS chunks and time to first canvas paint.
5. Open a factory line board. Record the extra JS chunks.

The login and home routes must not download the canvas or factories page
modules until the user opens those routes.

Production build on this branch (uncompressed / gzip):

- Entry `app-*.js`: about 2.7 MB / 783 kB. Home still loads this file.
- Lazy `factories-*.js`: about 594 kB / 162 kB after the user opens a workspace.
- Lazy `FactorySettingsRoutes-*.js`: about 68 kB / 17 kB after the user opens settings.
- Lazy `AppDefaultTabGate-*.js`: about 4 kB. Canvas page modules load after it.
- Lazy `AdminLayout-*.js`: about 4 kB.
- CSS `index-*.css`: about 667 kB / 92 kB.

`make check.build.ui` fails if the entry grows past 3.5 MB or if those lazy
chunks disappear.

## Canvas runtime

1. Open a live canvas that has an active run.
2. Confirm the websocket is open.
3. Confirm `ListRuns` does not poll every 60 seconds while the websocket is
   connected.
4. Record React commit count during 30 seconds of node executions.
5. Hide live activity or open the console tab. Confirm AppPage does not
   subscribe to node execution store updates.

## Factory and console

1. Scroll a line board with many cards. Record long tasks.
2. Open a console with several run widgets. Confirm widget page fetches stop
   near the eager page cap.

## Budgets

Use these as first targets after the route split:

- Home JS after login is smaller than the previous single bundle.
- Live canvas does not issue duplicate run polls while the websocket is healthy.
- Console and hidden live activity do not rebuild node runtime maps.
