import { showErrorToast } from "@/lib/toast";
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";

import {
  FLEET_REQUEST_TIMEOUT_MS,
  REFRESH_INTERVAL_MS,
  type BrokerTask,
  type FleetCapacity,
  type FleetRunner,
  type FleetTask,
  type InstallationFleet,
  type RecordPage,
  fetchBrokerTasks,
  fetchFleetDetails,
  fetchInstallationFleets,
  selectedFleetIdAfterRefresh,
} from "./fleetAdmin";

type ActivePoll = {
  generation: number;
  inFlight: boolean;
  abort: AbortController | null;
  timeout: number | null;
};

type DetailQuery = {
  fleetId: string;
  runnerCursor: string | null;
  taskCursor: string | null;
};

const createPoll = (): ActivePoll => ({
  generation: 0,
  inFlight: false,
  abort: null,
  timeout: null,
});

const stopPoll = (poll: ActivePoll) => {
  poll.generation += 1;
  poll.inFlight = false;
  poll.abort?.abort();
  poll.abort = null;
  if (poll.timeout !== null) {
    window.clearTimeout(poll.timeout);
    poll.timeout = null;
  }
};

const startPoll = (poll: ActivePoll) => {
  stopPoll(poll);
  const abort = new AbortController();
  const requestID = poll.generation;
  poll.abort = abort;
  poll.inFlight = true;
  poll.timeout = window.setTimeout(() => {
    if (poll.generation !== requestID) {
      return;
    }
    stopPoll(poll);
  }, FLEET_REQUEST_TIMEOUT_MS);
  return { requestID, signal: abort.signal };
};

const isCurrentPoll = (poll: ActivePoll, requestID: number) => poll.generation === requestID;

const finishPoll = (poll: ActivePoll, requestID: number) => {
  if (!isCurrentPoll(poll, requestID)) {
    return;
  }
  poll.inFlight = false;
  poll.abort = null;
  if (poll.timeout !== null) {
    window.clearTimeout(poll.timeout);
    poll.timeout = null;
  }
};

const sameDetailQuery = (left: DetailQuery, right: DetailQuery) =>
  left.fleetId === right.fleetId && left.runnerCursor === right.runnerCursor && left.taskCursor === right.taskCursor;

export const useAdminFleets = () => {
  const catalog = useFleetCatalog();
  const records = useFleetRecords(catalog.selectedFleetId);
  const broker = useBrokerTasks();
  return { ...catalog, ...records, ...broker };
};

const useFleetCatalog = () => {
  const [fleets, setFleets] = useState<InstallationFleet[] | null>(null);
  const [selectedFleetId, setSelectedFleetId] = useState<string | null>(null);
  const [fleetLoadFailed, setFleetLoadFailed] = useState(false);
  const selectedFleetIdRef = useRef<string | null>(null);
  const catalogPoll = useRef(createPoll()).current;

  const chooseFleet = useCallback((fleetId: string | null) => {
    if (selectedFleetIdRef.current === fleetId) {
      return;
    }
    selectedFleetIdRef.current = fleetId;
    setSelectedFleetId(fleetId);
  }, []);

  const loadFleets = useCallback(async () => {
    if (catalogPoll.inFlight) {
      return;
    }
    const { requestID, signal } = startPoll(catalogPoll);
    try {
      const nextFleets = await fetchInstallationFleets(signal);
      if (!isCurrentPoll(catalogPoll, requestID)) {
        return;
      }
      setFleetLoadFailed(false);
      setFleets(nextFleets);
      chooseFleet(selectedFleetIdAfterRefresh(selectedFleetIdRef.current, nextFleets));
    } catch (error) {
      if (!isCurrentPoll(catalogPoll, requestID) || signal.aborted) {
        return;
      }
      showErrorToast(error instanceof Error ? error.message : "Failed to load fleets");
      setFleetLoadFailed(true);
      setFleets((current) => current ?? []);
    } finally {
      finishPoll(catalogPoll, requestID);
    }
  }, [catalogPoll, chooseFleet]);

  useEffect(() => {
    void loadFleets();
    const interval = window.setInterval(() => void loadFleets(), REFRESH_INTERVAL_MS);
    return () => {
      window.clearInterval(interval);
      stopPoll(catalogPoll);
    };
  }, [catalogPoll, loadFleets]);

  return { fleets, selectedFleetId, fleetLoadFailed, chooseFleet };
};

const useFleetRecords = (selectedFleetId: string | null) => {
  const [capacity, setCapacity] = useState<FleetCapacity | null>(null);
  const [runnerCursors, setRunnerCursors] = useState<string[]>([]);
  const [taskCursors, setTaskCursors] = useState<string[]>([]);
  const [runners, setRunners] = useState<RecordPage<FleetRunner> | null>(null);
  const [tasks, setTasks] = useState<RecordPage<FleetTask> | null>(null);
  const [cursorFleetId, setCursorFleetId] = useState<string | null>(null);
  const detailsPoll = useRef(createPoll()).current;
  const detailsQuery = useRef<DetailQuery | null>(null);
  const selectedFleetIdRef = useRef(selectedFleetId);
  const runnerCursorRef = useRef<string | null>(null);
  const taskCursorRef = useRef<string | null>(null);
  selectedFleetIdRef.current = selectedFleetId;

  useLayoutEffect(() => {
    stopPoll(detailsPoll);
    detailsQuery.current = null;
  }, [detailsPoll, selectedFleetId]);

  useEffect(() => {
    setCursorFleetId(selectedFleetId);
    setRunnerCursors([]);
    setTaskCursors([]);
    setRunners(null);
    setTasks(null);
    setCapacity(null);
  }, [selectedFleetId]);

  const loadDetails = useCallback(
    async (fleetId: string, runnerCursor: string | null, taskCursor: string | null) => {
      const query = { fleetId, runnerCursor, taskCursor };
      if (detailsPoll.inFlight && detailsQuery.current && sameDetailQuery(detailsQuery.current, query)) {
        return;
      }
      const { requestID, signal } = startPoll(detailsPoll);
      detailsQuery.current = query;
      const matchesQuery = () =>
        isCurrentPoll(detailsPoll, requestID) &&
        fleetId === selectedFleetIdRef.current &&
        runnerCursor === runnerCursorRef.current &&
        taskCursor === taskCursorRef.current;
      try {
        const details = await fetchFleetDetails(fleetId, runnerCursor, taskCursor, signal);
        if (!matchesQuery()) {
          return;
        }
        setCapacity(details.capacity);
        setRunners(details.runners);
        setTasks(details.tasks);
      } catch (error) {
        if (!matchesQuery() || signal.aborted) {
          return;
        }
        showErrorToast(error instanceof Error ? error.message : "Failed to load fleet");
      } finally {
        finishPoll(detailsPoll, requestID);
      }
    },
    [detailsPoll],
  );

  const runnerCursor = runnerCursors.at(-1) ?? null;
  const taskCursor = taskCursors.at(-1) ?? null;
  runnerCursorRef.current = runnerCursor;
  taskCursorRef.current = taskCursor;
  const cursorsReady = cursorFleetId === selectedFleetId;

  useEffect(() => {
    if (!selectedFleetId || !cursorsReady) {
      return;
    }
    void loadDetails(selectedFleetId, runnerCursor, taskCursor);
    const interval = window.setInterval(() => {
      void loadDetails(selectedFleetId, runnerCursor, taskCursor);
    }, REFRESH_INTERVAL_MS);
    return () => {
      window.clearInterval(interval);
      stopPoll(detailsPoll);
      detailsQuery.current = null;
    };
  }, [cursorsReady, detailsPoll, loadDetails, runnerCursor, selectedFleetId, taskCursor]);

  return { capacity, runners, tasks, runnerCursors, taskCursors, setRunnerCursors, setTaskCursors };
};

const useBrokerTasks = () => {
  const [configured, setConfigured] = useState<boolean | null>(null);
  const [brokerTasks, setBrokerTasks] = useState<BrokerTask[]>([]);
  const [brokerLoaded, setBrokerLoaded] = useState(false);

  const loadBroker = useCallback(async () => {
    try {
      const data = await fetchBrokerTasks();
      setConfigured(data.configured);
      setBrokerTasks(data.tasks);
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to load runner tasks");
    } finally {
      setBrokerLoaded(true);
    }
  }, []);

  useEffect(() => {
    void loadBroker();
    const interval = window.setInterval(() => void loadBroker(), REFRESH_INTERVAL_MS);
    return () => window.clearInterval(interval);
  }, [loadBroker]);

  return { configured, brokerTasks, brokerLoaded };
};
