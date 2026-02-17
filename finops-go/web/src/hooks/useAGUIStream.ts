"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import type { UISchema } from "@/lib/types";

interface AGUIEvent {
  type: string;
  timestamp: string;
  workflow_id: string;
  data?: Record<string, unknown>;
}

interface StreamState {
  connected: boolean;
  phase: string;
  schema: UISchema | null;
  error: string | null;
  finished: boolean;
}

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

export function useAGUIStream(workflowId: string, enabled = true): StreamState {
  const [state, setState] = useState<StreamState>({
    connected: false,
    phase: "",
    schema: null,
    error: null,
    finished: false,
  });
  const eventSourceRef = useRef<EventSource | null>(null);

  const handleEvent = useCallback((eventType: string, data: string) => {
    try {
      const event: AGUIEvent = JSON.parse(data);

      switch (eventType) {
        case "STATE_SNAPSHOT": {
          const snapshotData = event.data as {
            phase: string;
            ui_schema: UISchema;
          };
          setState((prev) => ({
            ...prev,
            phase: snapshotData?.phase || "",
            schema: snapshotData?.ui_schema || null,
          }));
          break;
        }
        case "STATE_DELTA": {
          const deltaData = event.data as {
            phase: string;
            ui_schema: UISchema;
          };
          setState((prev) => ({
            ...prev,
            phase: deltaData?.phase || prev.phase,
            schema: deltaData?.ui_schema || prev.schema,
          }));
          break;
        }
        case "STEP_STARTED": {
          const stepData = event.data as { phase: string };
          setState((prev) => ({ ...prev, phase: stepData?.phase || "" }));
          break;
        }
        case "RUN_FINISHED":
          setState((prev) => ({ ...prev, finished: true }));
          break;
        case "RUN_ERROR": {
          const errorData = event.data as { message: string };
          setState((prev) => ({
            ...prev,
            error: errorData?.message || "Unknown error",
          }));
          break;
        }
      }
    } catch {
      // Ignore parse errors.
    }
  }, []);

  useEffect(() => {
    if (!enabled || !workflowId) return;

    const url = `${API_BASE}/api/v1/workflows/${encodeURIComponent(workflowId)}/stream`;
    const es = new EventSource(url);
    eventSourceRef.current = es;

    es.onopen = () => {
      setState((prev) => ({ ...prev, connected: true, error: null }));
    };

    es.onerror = () => {
      setState((prev) => ({
        ...prev,
        connected: false,
        error: "Connection lost",
      }));
    };

    // Listen for each AG-UI event type.
    const eventTypes = [
      "RUN_STARTED",
      "RUN_FINISHED",
      "RUN_ERROR",
      "STEP_STARTED",
      "STEP_FINISHED",
      "STATE_SNAPSHOT",
      "STATE_DELTA",
    ];
    for (const type of eventTypes) {
      es.addEventListener(type, (e: MessageEvent) => {
        handleEvent(type, e.data);
      });
    }

    return () => {
      es.close();
      eventSourceRef.current = null;
    };
  }, [workflowId, enabled, handleEvent]);

  return state;
}
