"use client";

import { use } from "react";
import Link from "next/link";
import { useWorkflowState } from "@/hooks/useWorkflowState";
import { useAGUIStream } from "@/hooks/useAGUIStream";
import { ComponentRenderer } from "@/components/schema/ComponentRenderer";
import { approveWorkflow, denyWorkflow } from "@/lib/api";
import { useState } from "react";

export default function WorkflowPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const { result, schema, error, loading } = useWorkflowState(id);
  const stream = useAGUIStream(id, !result?.state.should_terminate);
  const [actionError, setActionError] = useState<string | null>(null);
  const [actionPending, setActionPending] = useState(false);

  // Prefer streamed schema over initial fetch.
  const activeSchema = stream.schema || schema;

  if (loading) return <p className="text-gray-500">Loading workflow...</p>;
  if (error) return <p className="text-red-600">Error: {error}</p>;
  if (!result || !activeSchema) return <p className="text-gray-500">No data</p>;

  const state = result.state;
  const isPending = state.approval === "pending" && state.analysis;

  async function handleApprove() {
    setActionPending(true);
    try {
      await approveWorkflow(id, "ui-user");
      window.location.reload();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Failed");
    } finally {
      setActionPending(false);
    }
  }

  async function handleDeny() {
    setActionPending(true);
    try {
      await denyWorkflow(id, "ui-user", "denied via UI");
      window.location.reload();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Failed");
    } finally {
      setActionPending(false);
    }
  }

  return (
    <div>
      <Link href="/" className="text-blue-600 text-sm hover:underline">
        &larr; Back to inbox
      </Link>

      <div className="flex items-center justify-between mt-4 mb-4">
        <h2 className="text-lg font-semibold font-mono">{id}</h2>
        <div className="flex items-center gap-2 text-sm">
          <span className="text-gray-500">Phase:</span>
          <span className="font-medium">
            {stream.phase || state.current_phase}
          </span>
          {stream.connected && (
            <span className="w-2 h-2 rounded-full bg-green-500" title="Live" />
          )}
        </div>
      </div>

      <ComponentRenderer schema={activeSchema} />

      {isPending && (
        <div className="mt-4 flex gap-3">
          <button
            onClick={handleApprove}
            disabled={actionPending}
            className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50"
          >
            Approve
          </button>
          <button
            onClick={handleDeny}
            disabled={actionPending}
            className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 disabled:opacity-50"
          >
            Deny
          </button>
        </div>
      )}

      {actionError && (
        <p className="mt-2 text-red-600 text-sm">{actionError}</p>
      )}

      {result.reason && (
        <div className="mt-4 text-sm text-gray-500">
          Workflow result: <span className="font-medium">{result.reason}</span>
        </div>
      )}
    </div>
  );
}
