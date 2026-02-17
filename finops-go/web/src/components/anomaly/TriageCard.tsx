import type { UIComponent } from "@/lib/types";

const severityColors: Record<string, string> = {
  low: "bg-green-100 text-green-800",
  medium: "bg-yellow-100 text-yellow-800",
  high: "bg-orange-100 text-orange-800",
  critical: "bg-red-100 text-red-800",
};

export function TriageCard({ component }: { component: UIComponent }) {
  const { data } = component;
  const severity = String(data?.severity ?? "unknown");
  const colorClass = severityColors[severity] || "bg-gray-100 text-gray-800";

  return (
    <section className="border rounded-lg p-4">
      <h2 className="text-lg font-semibold mb-2">{component.title}</h2>
      <div className="flex items-center gap-3 mb-2">
        <span className={`px-2 py-1 rounded text-xs font-medium ${colorClass}`}>
          {severity}
        </span>
        <span className="text-sm text-gray-600">
          {String(data?.category ?? "").replace(/_/g, " ")}
        </span>
        <span className="text-sm text-gray-400">
          {(Number(data?.confidence ?? 0) * 100).toFixed(0)}% confidence
        </span>
      </div>
      {data?.summary && (
        <p className="text-sm text-gray-700">{String(data.summary)}</p>
      )}
    </section>
  );
}
