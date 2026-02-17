// API client for the FinOps backend.

import type { WorkflowSummary, WorkflowResult, UISchema } from "./types";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, init);
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`API error ${res.status}: ${body}`);
  }
  return res.json();
}

export function listWorkflows(): Promise<WorkflowSummary[]> {
  return fetchJSON("/api/v1/workflows");
}

export function getWorkflowState(id: string): Promise<WorkflowResult> {
  return fetchJSON(`/api/v1/workflows/${encodeURIComponent(id)}`);
}

export function getWorkflowUI(id: string): Promise<UISchema> {
  return fetchJSON(`/api/v1/workflows/${encodeURIComponent(id)}/ui`);
}

export function approveWorkflow(
  id: string,
  by: string
): Promise<{ result: string }> {
  return fetchJSON(`/api/v1/workflows/${encodeURIComponent(id)}/approve`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ by }),
  });
}

export function denyWorkflow(
  id: string,
  by: string,
  reason?: string
): Promise<{ result: string }> {
  return fetchJSON(`/api/v1/workflows/${encodeURIComponent(id)}/deny`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ by, reason }),
  });
}
