import { showErrorToast, showSuccessToast } from "@/lib/toast";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type Dispatch,
  type MutableRefObject,
  type SetStateAction,
} from "react";
import {
  activatePriceBook,
  fetchPriceBooks,
  savePriceBooks,
  syncPriceBooks,
  type PriceBookModelRate,
  type PriceBooksResponse,
  type PriceBookVMRate,
} from "./priceBooksApi";

export function usePriceBookCatalog() {
  const [data, setData] = useState<PriceBooksResponse | null>(null);
  const [models, setModels] = useState<PriceBookModelRate[]>([]);
  const [vms, setVMs] = useState<PriceBookVMRate[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadFailed, setLoadFailed] = useState(false);
  const [versionLoading, setVersionLoading] = useState(false);
  const [pendingVersion, setPendingVersion] = useState<string | undefined>();
  const loadAbort = useRef<AbortController | null>(null);
  const loadGeneration = useRef(0);

  const applyCatalog = useCallback((payload: PriceBooksResponse) => {
    setData(payload);
    setPendingVersion(payload.version);
    setModels(payload.models.map((rate) => ({ ...rate })));
    setVMs(payload.vms.map((rate) => ({ ...rate })));
  }, []);

  const supersedeLoads = useCallback(() => {
    loadAbort.current?.abort();
    loadAbort.current = null;
    loadGeneration.current += 1;
    setVersionLoading(false);
  }, []);

  const loadPriceBooks = useCallback(
    async (version?: string) => {
      await loadCatalogVersion({
        version,
        loadAbort,
        loadGeneration,
        applyCatalog,
        setLoading,
        setLoadFailed,
        setData,
        setVersionLoading,
        setPendingVersion,
      });
    },
    [applyCatalog],
  );

  useEffect(() => {
    void loadPriceBooks();
    const abortRef = loadAbort;
    return () => {
      abortRef.current?.abort();
    };
  }, [loadPriceBooks]);

  return {
    data,
    models,
    vms,
    setModels,
    setVMs,
    loading,
    loadFailed,
    versionLoading,
    pendingVersion,
    applyCatalog,
    loadPriceBooks,
    supersedeLoads,
  };
}

async function loadCatalogVersion({
  version,
  loadAbort,
  loadGeneration,
  applyCatalog,
  setLoading,
  setLoadFailed,
  setData,
  setVersionLoading,
  setPendingVersion,
}: {
  version?: string;
  loadAbort: MutableRefObject<AbortController | null>;
  loadGeneration: MutableRefObject<number>;
  applyCatalog: (payload: PriceBooksResponse) => void;
  setLoading: (value: boolean) => void;
  setLoadFailed: (value: boolean) => void;
  setData: (value: PriceBooksResponse | null) => void;
  setVersionLoading: (value: boolean) => void;
  setPendingVersion: (value: string | undefined) => void;
}) {
  const isFirstLoad = version === undefined;
  if (isFirstLoad) {
    setLoading(true);
    setLoadFailed(false);
  } else {
    setPendingVersion(version);
    setVersionLoading(true);
  }

  loadAbort.current?.abort();
  const controller = new AbortController();
  loadAbort.current = controller;
  const generation = ++loadGeneration.current;

  try {
    const payload = await fetchPriceBooks(version, controller.signal);
    if (generation !== loadGeneration.current) {
      return;
    }

    applyCatalog(payload);
    setLoadFailed(false);
  } catch (error) {
    if (generation !== loadGeneration.current || controller.signal.aborted) {
      return;
    }

    showErrorToast(error instanceof Error ? error.message : "Failed to load price books");
    if (isFirstLoad) {
      setLoadFailed(true);
      setData(null);
    } else {
      setPendingVersion(undefined);
    }
  } finally {
    if (generation !== loadGeneration.current) {
      return;
    }
    if (isFirstLoad) {
      setLoading(false);
    } else {
      setVersionLoading(false);
    }
  }
}

export function usePriceBookEdits(
  models: PriceBookModelRate[],
  vms: PriceBookVMRate[],
  setModels: Dispatch<SetStateAction<PriceBookModelRate[]>>,
  setVMs: Dispatch<SetStateAction<PriceBookVMRate[]>>,
  applyCatalog: (payload: PriceBooksResponse) => void,
  version: string,
  supersedeLoads: () => void,
) {
  const [saving, setSaving] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [activating, setActivating] = useState(false);
  const modelKeys = useMemo(() => new Set(models.map((rate) => `${rate.match_key}:${rate.match_mode}`)), [models]);
  const vmKeys = useMemo(() => new Set(vms.map((rate) => `${rate.match_key}:${rate.match_mode}`)), [vms]);

  return {
    saving,
    syncing,
    activating,
    handleAddModel: (rate: PriceBookModelRate) => appendUniqueModel(models, modelKeys, rate, setModels),
    handleAddVM: (rate: PriceBookVMRate) => appendUniqueVM(vms, vmKeys, rate, setVMs),
    handleSave: () => void saveCurrentRates(version, models, vms, applyCatalog, supersedeLoads, setSaving),
    handleSync: () => void syncCurrentRates(applyCatalog, supersedeLoads, setSyncing),
    handleActivate: (targetVersion: string, onDone: () => void) =>
      void activateCurrentVersion(targetVersion, applyCatalog, supersedeLoads, setActivating, onDone),
    handleModelChange: (index: number, patch: Partial<PriceBookModelRate>) => {
      setModels((current) => current.map((rate, rateIndex) => (rateIndex === index ? { ...rate, ...patch } : rate)));
    },
    handleVMChange: (index: number, micros: number) => {
      setVMs((current) =>
        current.map((rate, rateIndex) => (rateIndex === index ? { ...rate, micros_per_second: micros } : rate)),
      );
    },
  };
}

function appendUniqueModel(
  models: PriceBookModelRate[],
  keys: Set<string>,
  rate: PriceBookModelRate,
  setModels: Dispatch<SetStateAction<PriceBookModelRate[]>>,
): boolean {
  const key = rate.match_key.trim().toLowerCase();
  if (key === "") {
    showErrorToast("Enter a match key.");
    return false;
  }
  if (keys.has(`${key}:${rate.match_mode}`)) {
    showErrorToast("That model rate already exists.");
    return false;
  }
  setModels([...models, { ...rate, match_key: key }]);
  return true;
}

function appendUniqueVM(
  vms: PriceBookVMRate[],
  keys: Set<string>,
  rate: PriceBookVMRate,
  setVMs: Dispatch<SetStateAction<PriceBookVMRate[]>>,
): boolean {
  const key = rate.match_key.trim().toLowerCase();
  if (key === "") {
    showErrorToast("Enter a machine type.");
    return false;
  }
  if (keys.has(`${key}:${rate.match_mode}`)) {
    showErrorToast("That VM rate already exists.");
    return false;
  }
  setVMs([...vms, { ...rate, match_key: key }]);
  return true;
}

async function saveCurrentRates(
  version: string,
  models: PriceBookModelRate[],
  vms: PriceBookVMRate[],
  applyCatalog: (payload: PriceBooksResponse) => void,
  supersedeLoads: () => void,
  setSaving: (value: boolean) => void,
) {
  setSaving(true);
  supersedeLoads();
  try {
    applyCatalog(await savePriceBooks(version, models, vms));
    showSuccessToast("Saved a new current price book.");
  } catch (error) {
    showErrorToast(error instanceof Error ? error.message : "Failed to save price books");
  } finally {
    setSaving(false);
  }
}

async function syncCurrentRates(
  applyCatalog: (payload: PriceBooksResponse) => void,
  supersedeLoads: () => void,
  setSyncing: (value: boolean) => void,
) {
  setSyncing(true);
  supersedeLoads();
  try {
    const payload = await syncPriceBooks();
    applyCatalog(payload);
    showSuccessToast(`Updated ${payload.updated_count} model rates and added ${payload.added_count}.`);
  } catch (error) {
    showErrorToast(error instanceof Error ? error.message : "Failed to update model rates");
  } finally {
    setSyncing(false);
  }
}

async function activateCurrentVersion(
  version: string,
  applyCatalog: (payload: PriceBooksResponse) => void,
  supersedeLoads: () => void,
  setActivating: (value: boolean) => void,
  onDone: () => void,
) {
  setActivating(true);
  supersedeLoads();
  try {
    applyCatalog(await activatePriceBook(version));
    onDone();
    showSuccessToast("This version is now current.");
  } catch (error) {
    showErrorToast(error instanceof Error ? error.message : "Failed to switch price book");
  } finally {
    setActivating(false);
  }
}
