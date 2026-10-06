package browser

import (
	"net/url"
	"os"
	"testing"

	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
)

func TestMonacoProductionWorkerStarts(t *testing.T) {
	pageURL := os.Getenv("MONACO_PAGE_URL")
	workerURL := os.Getenv("MONACO_WORKER_URL")
	if pageURL == "" || workerURL == "" {
		t.Fatal("MONACO_PAGE_URL and MONACO_WORKER_URL are required")
	}

	pageOrigin := originOf(t, pageURL)
	workerOrigin := originOf(t, workerURL)
	require.NotEqual(t, pageOrigin, workerOrigin)

	runner, err := pw.Run()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, runner.Stop())
	})

	browser, err := runner.Chromium.Launch()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, browser.Close())
	})

	page, err := browser.NewPage()
	require.NoError(t, err)

	_, err = page.Goto(pageURL)
	require.NoError(t, err)

	result, err := page.Evaluate(monacoWorkerStartupScript, workerURL)
	require.NoError(t, err)
	require.Equal(t, "started", result)
}

func originOf(t *testing.T, raw string) string {
	t.Helper()
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	return parsed.Scheme + "://" + parsed.Host
}

const monacoWorkerStartupScript = `(scriptUrl) => {
  const source = "importScripts(" + JSON.stringify(scriptUrl) + "); postMessage('started');";
  const blob = new Blob([source], { type: "text/javascript" });
  const worker = new Worker(URL.createObjectURL(blob));
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      worker.terminate();
      reject(new Error("worker did not start"));
    }, 20000);
    worker.onmessage = (event) => {
      if (event.data !== "started") {
        return;
      }
      clearTimeout(timer);
      worker.terminate();
      resolve("started");
    };
    worker.onerror = (event) => {
      clearTimeout(timer);
      worker.terminate();
      reject(new Error(event.message || "worker failed to start"));
    };
  });
}`
