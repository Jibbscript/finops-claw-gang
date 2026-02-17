"use client";

import { useEffect, useState } from "react";
import type { WorkflowResult, UISchema } from "@/lib/types";
import { getWorkflowState, getWorkflowUI } from "@/lib/api";

export function useWorkflowState(id: string) {
  const [result, setResult] = useState<WorkflowResult | null>(null);
  const [schema, setSchema] = useState<UISchema | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const [r, s] = await Promise.all([
          getWorkflowState(id),
          getWorkflowUI(id),
        ]);
        if (!cancelled) {
          setResult(r);
          setSchema(s);
          setLoading(false);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Unknown error");
          setLoading(false);
        }
      }
    }

    load();
    return () => {
      cancelled = true;
    };
  }, [id]);

  return { result, schema, error, loading };
}
