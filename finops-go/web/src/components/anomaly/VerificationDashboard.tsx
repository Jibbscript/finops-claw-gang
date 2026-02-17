import type { UIComponent } from "@/lib/types";

export function VerificationDashboard({
  component,
}: {
  component: UIComponent;
}) {
  const { data } = component;
  return (
    <section className="border rounded-lg p-4">
      <h2 className="text-lg font-semibold mb-2">{component.title}</h2>
      <div className="grid grid-cols-2 gap-2 text-sm">
        <div>
          <span className="text-gray-500">Cost Reduction:</span>{" "}
          <span
            className={
              data?.cost_reduction_observed ? "text-green-600" : "text-red-600"
            }
          >
            {data?.cost_reduction_observed ? "Yes" : "No"}
          </span>
        </div>
        <div>
          <span className="text-gray-500">Health:</span>{" "}
          <span
            className={
              data?.service_health_ok ? "text-green-600" : "text-red-600"
            }
          >
            {data?.service_health_ok ? "OK" : "Degraded"}
          </span>
        </div>
        {data?.observed_savings_daily && (
          <div>
            <span className="text-gray-500">Daily Savings:</span>{" "}
            <span className="font-mono text-green-600">
              ${Number(data.observed_savings_daily).toFixed(2)}
            </span>
          </div>
        )}
        <div>
          <span className="text-gray-500">Recommendation:</span>{" "}
          <span className="font-medium">
            {String(data?.recommendation ?? "").replace(/_/g, " ")}
          </span>
        </div>
      </div>
    </section>
  );
}
