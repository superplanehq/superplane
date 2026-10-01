export type Cancelable<T> = Promise<T> & { cancel: () => void };

const CANCELLATION = { type: "cancelation" };

export function createLocalMonacoInit(
  init: () => Cancelable<unknown>,
  loadLocalMonaco: () => Promise<void>,
): () => Cancelable<unknown> {
  let localMonaco: Promise<void> | undefined;

  return () => {
    let cancelled = false;
    let cancelPending = () => {
      cancelled = true;
    };

    const ready = (localMonaco ??= loadLocalMonaco().then(
      () => undefined,
      (error: unknown) => {
        localMonaco = undefined;
        throw error;
      },
    ));

    const promise = ready.then(() => {
      if (cancelled) {
        throw CANCELLATION;
      }

      const editor = init();
      cancelPending = () => {
        cancelled = true;
        editor.cancel();
      };
      return editor;
    }) as Cancelable<unknown>;

    promise.cancel = () => {
      cancelPending();
    };

    return promise;
  };
}
