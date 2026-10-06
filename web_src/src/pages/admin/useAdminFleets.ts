import { showErrorToast } from "@/lib/toast";
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";

import {
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

  const chooseFleet = useCallback((fleetId: string | null) => {
    if (selectedFleetIdRef.current === fleetId) {
      return;
    }
    selectedFleetIdRef.current = fleetId;
    setSelectedFleetId(fleetId);
  }, []);

  const catalogRequest = useRef(0);
  const catalogInFlight = useRef(false);

  const loadFleets = useCallback(async () => {
    if (catalogInFlight.current) {
      return;
    }
    catalogInFlight.current = true;
    const requestID = ++catalogRequest.current;
    try {
      const nextFleets = await fetchInstallationFleets();
      if (requestID !== catalogRequest.current) {
        return;
      }
      setFleetLoadFailed(false);
      setFleets(nextFleets);
      chooseFleet(selectedFleetIdAfterRefresh(selectedFleetIdRef.current, nextFleets));
    } catch (error) {
      if (requestID !== catalogRequest.current) {
        return;
      }
      showErrorToast(error instanceof Error ? error.message : "Failed to load fleets");
      setFleetLoadFailed(true);
      setFleets((current) => current ?? []);
    } finally {
      if (requestID === catalogRequest.current) {
        catalogInFlight.current = false;
      }
    }
  }, [chooseFleet]);

  useEffect(() => {
    void loadFleets();
    const interval = window.setInterval(() => void loadFleets(), REFRESH_INTERVAL_MS);
    return () => window.clearInterval(interval);
  }, [loadFleets]);

  return { fleets, selectedFleetId, fleetLoadFailed, chooseFleet };
};

const useFleetRecords = (selectedFleetId: string | null) => {
  const [capacity, setCapacity] = useState<FleetCapacity | null>(null);
  const [runnerCursors, setRunnerCursors] = useState<string[]>([]);
  const [taskCursors, setTaskCursors] = useState<string[]>([]);
  const [runners, setRunners] = useState<RecordPage<FleetRunner> | null>(null);
  const [tasks, setTasks] = useState<RecordPage<FleetTask> | null>(null);
  const [cursorFleetId, setCursorFleetId] = useState<string | null>(null);
  const detailsRequest = useRef(0);
  const detailsInFlight = useRef(false);
  const selectedFleetIdRef = useRef(selectedFleetId);
  selectedFleetIdRef.current = selectedFleetId;

  useLayoutEffect(() => {
    detailsRequest.current += 1;
    detailsInFlight.current = false;
  }, [selectedFleetId]);

  useEffect(() => {
    setCursorFleetId(selectedFleetId);
    setRunnerCursors([]);
    setTaskCursors([]);
    setRunners(null);
    setTasks(null);
    setCapacity(null);
  }, [selectedFleetId]);

  const loadDetails = useCallback(async (fleetId: string, runnerCursor: string | null, taskCursor: string | null) => {
    if (detailsInFlight.current) {
      return;
    }
    detailsInFlight.current = true;
    const requestID = ++detailsRequest.current;
    try {
      const details = await fetchFleetDetails(fleetId, runnerCursor, taskCursor);
      if (requestID !== detailsRequest.current || fleetId !== selectedFleetIdRef.current) {
        return;
      }
      setCapacity(details.capacity);
      setRunners(details.runners);
      setTasks(details.tasks);
    } catch (error) {
      if (requestID !== detailsRequest.current || fleetId !== selectedFleetIdRef.current) {
        return;
      }
      showErrorToast(error instanceof Error ? error.message : "Failed to load fleet");
    } finally {
      if (requestID === detailsRequest.current) {
        detailsInFlight.current = false;
      }
    }
  }, []);

  const runnerCursor = runnerCursors.at(-1) ?? null;
  const taskCursor = taskCursors.at(-1) ?? null;
  const cursorsReady = cursorFleetId === selectedFleetId;

  useEffect(() => {
    if (!selectedFleetId || !cursorsReady) {
      return;
    }
    void loadDetails(selectedFleetId, runnerCursor, taskCursor);
    const interval = window.setInterval(() => {
      void loadDetails(selectedFleetId, runnerCursor, taskCursor);
    }, REFRESH_INTERVAL_MS);
    return () => window.clearInterval(interval);
  }, [cursorsReady, loadDetails, runnerCursor, selectedFleetId, taskCursor]);

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
