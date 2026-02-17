import type { UIComponent } from "@/lib/types";

const riskColors: Record<string, string> = {
  low: "text-green-600",
  low_medium: "text-yellow-600",
  medium: "text-yellow-600",
  high: "text-orange-600",
  critical: "text-red-600",
};

export function ActionPlan({ component }: { component: UIComponent }) {
  const { data } = component;
  const actions = (data?.actions as Array<Record<string, unknown>>) || [];

  return (
    <section className="border rounded-lg p-4">
      <h2 className="text-lg font-semibold mb-2">{component.title}</h2>
      {data?.root_cause && (
        <p className="text-sm text-gray-700 mb-3">{String(data.root_cause)}</p>
      )}
      {actions.length > 0 && (
        <div className="space-y-2">
          {actions.map((action, i) => (
            <div
              key={String(action.action_id || i)}
              className="flex items-center justify-between bg-gray-50 rounded p-2 text-sm"
            >
              <div>
                <span className="font-medium">
                  {String(action.description || "")}
                </span>
                <span className="ml-2 text-xs text-gray-400">
                  {String(action.action_type || "")}
                </span>
              </div>
              <div className="flex items-center gap-2">
                {action.savings && (
                  <span className="text-green-600 font-mono text-xs">
                    -${Number(action.savings).toFixed(0)}/mo
                  </span>
                )}
                <span
                  className={`text-xs font-medium ${riskColors[String(action.risk_level)] || ""}`}
                >
                  {String(action.risk_level || "")}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
      {data?.action_type && !actions.length && (
        <div className="bg-gray-50 rounded p-2 text-sm">
          <pre className="text-xs overflow-x-auto">
            {JSON.stringify(data, null, 2)}
          </pre>
        </div>
      )}
    </section>
  );
}
