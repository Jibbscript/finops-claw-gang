import type { UIComponent } from "@/lib/types";

export function AnomalySummary({ component }: { component: UIComponent }) {
  const { data } = component;
  return (
    <section className="border rounded-lg p-4">
      <h2 className="text-lg font-semibold mb-2">{component.title}</h2>
      <div className="grid grid-cols-2 gap-2 text-sm">
        <div>
          <span className="text-gray-500">Service:</span>{" "}
          {String(data?.service ?? "")}
        </div>
        <div>
          <span className="text-gray-500">Account:</span>{" "}
          {String(data?.account_id ?? "")}
        </div>
        <div>
          <span className="text-gray-500">Delta:</span>{" "}
          <span className="font-mono text-red-600">
            +${Number(data?.delta_dollars ?? 0).toFixed(2)}/day
          </span>
        </div>
        <div>
          <span className="text-gray-500">Change:</span>{" "}
          {Number(data?.delta_percent ?? 0).toFixed(1)}%
        </div>
        <div className="col-span-2 text-xs text-gray-400">
          Detected: {String(data?.detected_at ?? "")}
        </div>
      </div>
    </section>
  );
}
