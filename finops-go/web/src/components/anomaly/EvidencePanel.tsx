import type { UIComponent } from "@/lib/types";

export function EvidencePanel({ component }: { component: UIComponent }) {
  const { data } = component;
  return (
    <section className="border rounded-lg p-4">
      <h2 className="text-lg font-semibold mb-2">{component.title}</h2>
      <pre className="text-xs bg-gray-50 p-3 rounded overflow-x-auto">
        {JSON.stringify(data, null, 2)}
      </pre>
    </section>
  );
}
