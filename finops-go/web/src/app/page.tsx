"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import type { WorkflowSummary } from "@/lib/types";
import { listWorkflows } from "@/lib/api";

const statusColors: Record<string, string> = {
  WORKFLOW_EXECUTION_STATUS_RUNNING: "bg-blue-100 text-blue-800",
  WORKFLOW_EXECUTION_STATUS_COMPLETED: "bg-green-100 text-green-800",
  Running: "bg-blue-100 text-blue-800",
  Completed: "bg-green-100 text-green-800",
};

export default function InboxPage() {
  const [workflows, setWorkflows] = useState<WorkflowSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listWorkflows()
      .then(setWorkflows)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <p className="text-gray-500">Loading workflows...</p>;
  if (error) return <p className="text-red-600">Error: {error}</p>;

  return (
    <div>
      <h2 className="text-lg font-semibold mb-4">Anomaly Inbox</h2>
      {workflows.length === 0 ? (
        <p className="text-gray-500">No anomaly workflows found.</p>
      ) : (
        <div className="space-y-2">
          {workflows.map((wf) => (
            <Link
              key={wf.workflow_id}
              href={`/workflows/${encodeURIComponent(wf.workflow_id)}`}
              className="block border rounded-lg p-3 hover:bg-gray-50 transition-colors"
            >
              <div className="flex items-center justify-between">
                <span className="font-mono text-sm">{wf.workflow_id}</span>
                <span
                  className={`px-2 py-0.5 rounded text-xs font-medium ${statusColors[wf.status] || "bg-gray-100"}`}
                >
                  {wf.status.replace("WORKFLOW_EXECUTION_STATUS_", "")}
                </span>
              </div>
              <div className="text-xs text-gray-400 mt-1">
                Started: {new Date(wf.start_time).toLocaleString()}
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
