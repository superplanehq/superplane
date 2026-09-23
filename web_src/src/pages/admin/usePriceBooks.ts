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
  deletePriceBook,
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

type CatalogLoaders = {
  version?: string;
  loadAbort: MutableRefObject<AbortController | null>;
  loadGeneration: MutableRefObject<number>;
  applyCatalog: (payload: PriceBooksResponse) => void;
  setLoading: (value: boolean) => void;
  setLoadFailed: (value: boolean) => void;
  setData: (value: PriceBooksResponse | null) => void;
  setVersionLoading: (value: boolean) => void;
  setPendingVersion: (value: string | undefined) => void;
};

async function loadCatalogVersion(loaders: CatalogLoaders) {
  const isFirstLoad = loaders.version === undefined;
  startCatalogLoad(loaders, isFirstLoad);

  loaders.loadAbort.current?.abort();
  const controller = new AbortController();
  loaders.loadAbort.current = controller;
  const generation = ++loaders.loadGeneration.current;

  try {
    const payload = await fetchPriceBooks(loaders.version, controller.signal);
    if (generation !== loaders.loadGeneration.current) {
      return;
    }

    loaders.applyCatalog(payload);
    loaders.setLoadFailed(false);
  } catch (error) {
    handleCatalogLoadError(loaders, isFirstLoad, generation, controller, error);
  } finally {
    if (generation === loaders.loadGeneration.current) {
      finishCatalogLoad(loaders, isFirstLoad);
    }
  }
}

function startCatalogLoad(loaders: CatalogLoaders, isFirstLoad: boolean) {
  if (isFirstLoad) {
    loaders.setLoading(true);
    loaders.setLoadFailed(false);
    return;
  }
  loaders.setPendingVersion(loaders.version);
  loaders.setVersionLoading(true);
}

function finishCatalogLoad(loaders: CatalogLoaders, isFirstLoad: boolean) {
  if (isFirstLoad) {
    loaders.setLoading(false);
    return;
  }
  loaders.setVersionLoading(false);
}

function handleCatalogLoadError(
  loaders: CatalogLoaders,
  isFirstLoad: boolean,
  generation: number,
  controller: AbortController,
  error: unknown,
) {
  if (generation !== loaders.loadGeneration.current || controller.signal.aborted) {
    return;
  }

  showErrorToast(error instanceof Error ? error.message : "Failed to load price books");
  if (isFirstLoad) {
    loaders.setLoadFailed(true);
    loaders.setData(null);
    return;
  }
  loaders.setPendingVersion(undefined);
}

type PriceBookEditCatalog = {
  models: PriceBookModelRate[];
  vms: PriceBookVMRate[];
  setModels: Dispatch<SetStateAction<PriceBookModelRate[]>>;
  setVMs: Dispatch<SetStateAction<PriceBookVMRate[]>>;
  applyCatalog: (payload: PriceBooksResponse) => void;
  data: PriceBooksResponse | null;
  supersedeLoads: () => void;
};

export function usePriceBookEdits(catalog: PriceBookEditCatalog) {
  const [saving, setSaving] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [activating, setActivating] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const vmKeys = useMemo(
    () => new Set(catalog.vms.map((rate) => `${rate.match_key}:${rate.match_mode}`)),
    [catalog.vms],
  );

  return {
    saving,
    syncing,
    activating,
    deleting,
    handleAddVM: (rate: PriceBookVMRate) => appendUniqueVM(catalog.vms, vmKeys, rate, catalog.setVMs),
    handleSave: () => void saveCurrentRates(catalog, setSaving),
    handleSync: (provider: string) => void syncCurrentRates(catalog, provider, setSyncing),
    handleActivate: (targetVersion: string, onDone: () => void) =>
      void activateCurrentVersion(targetVersion, catalog, setActivating, onDone),
    handleDelete: (targetVersion: string, onDone: () => void) =>
      void deleteCurrentVersion(targetVersion, catalog, setDeleting, onDone),
    handleRemoveVM: (index: number) => {
      catalog.setVMs((current) => current.filter((_, rateIndex) => rateIndex !== index));
    },
  };
}

function normalizeVMMatchKey(matchKey: string): string | undefined {
  const key = matchKey.trim().toLowerCase();
  if (key === "") {
    showErrorToast("Enter a machine type.");
    return undefined;
  }
  return key;
}

function appendUniqueVM(
  vms: PriceBookVMRate[],
  keys: Set<string>,
  rate: PriceBookVMRate,
  setVMs: Dispatch<SetStateAction<PriceBookVMRate[]>>,
): boolean {
  const key = normalizeVMMatchKey(rate.match_key);
  if (key === undefined) {
    return false;
  }
  if (keys.has(`${key}:${rate.match_mode}`)) {
    showErrorToast("That machine rate already exists.");
    return false;
  }
  setVMs([...vms, { ...rate, match_key: key }]);
  return true;
}

async function saveCurrentRates(catalog: PriceBookEditCatalog, setSaving: (value: boolean) => void) {
  setSaving(true);
  catalog.supersedeLoads();
  try {
    catalog.applyCatalog(await savePriceBooks(catalog.data?.version ?? "", catalog.models, catalog.vms));
    showSuccessToast("Saved a new current price book.");
  } catch (error) {
    showErrorToast(error instanceof Error ? error.message : "Failed to save price books");
  } finally {
    setSaving(false);
  }
}

async function syncCurrentRates(catalog: PriceBookEditCatalog, provider: string, setSyncing: (value: boolean) => void) {
  setSyncing(true);
  catalog.supersedeLoads();
  try {
    const payload = await syncPriceBooks(provider);
    catalog.applyCatalog(payload);
    showSuccessToast(`Updated ${payload.updated_count} model rates and added ${payload.added_count}.`);
  } catch (error) {
    showErrorToast(error instanceof Error ? error.message : "Failed to update model rates");
  } finally {
    setSyncing(false);
  }
}

async function deleteCurrentVersion(
  version: string,
  catalog: PriceBookEditCatalog,
  setDeleting: (value: boolean) => void,
  onDone: () => void,
) {
  setDeleting(true);
  catalog.supersedeLoads();
  try {
    catalog.applyCatalog(await deletePriceBook(version));
    onDone();
    showSuccessToast("Deleted this price book version.");
  } catch (error) {
    showErrorToast(error instanceof Error ? error.message : "Failed to delete price book");
  } finally {
    setDeleting(false);
  }
}

async function activateCurrentVersion(
  version: string,
  catalog: PriceBookEditCatalog,
  setActivating: (value: boolean) => void,
  onDone: () => void,
) {
  setActivating(true);
  catalog.supersedeLoads();
  try {
    catalog.applyCatalog(await activatePriceBook(version));
    onDone();
    showSuccessToast("This version is now current.");
  } catch (error) {
    showErrorToast(error instanceof Error ? error.message : "Failed to switch price book");
  } finally {
    setActivating(false);
  }
}
