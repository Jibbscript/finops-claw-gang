import type { UIComponent } from "@/lib/types";

export function ExecutionResults({ component }: { component: UIComponent }) {
  const { data } = component;
  const results = (data?.results as Array<Record<string, unknown>>) || [];

  return (
    <section className="border rounded-lg p-4">
      <h2 className="text-lg font-semibold mb-2">{component.title}</h2>
      <div className="space-y-2">
        {results.map((r, i) => (
          <div
            key={String(r.action_id || i)}
            className="flex items-center justify-between bg-gray-50 rounded p-2 text-sm"
          >
            <div className="flex items-center gap-2">
              <span className={r.success ? "text-green-600" : "text-red-600"}>
                {r.success ? "Success" : "Failed"}
              </span>
              <span className="text-gray-600">{String(r.details || "")}</span>
            </div>
            <span className="text-xs text-gray-400">
              {String(r.executed_at || "")}
            </span>
          </div>
        ))}
      </div>
    </section>
  );
}
