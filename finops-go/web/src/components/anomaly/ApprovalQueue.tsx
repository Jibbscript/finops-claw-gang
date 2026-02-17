import type { UIComponent } from "@/lib/types";

export function ApprovalQueue({ component }: { component: UIComponent }) {
  const { data } = component;
  return (
    <section className="border-2 border-yellow-300 rounded-lg p-4 bg-yellow-50">
      <h2 className="text-lg font-semibold mb-2">{component.title}</h2>
      <div className="text-sm">
        <span className="font-medium">Status:</span>{" "}
        <span className="text-yellow-700">
          {String(data?.approval_status ?? "pending")}
        </span>
      </div>
      {data?.approval_details && (
        <p className="text-sm text-gray-600 mt-1">
          {String(data.approval_details)}
        </p>
      )}
    </section>
  );
}
