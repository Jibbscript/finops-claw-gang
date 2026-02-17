"use client";

import type { UISchema } from "@/lib/types";
import { componentRegistry } from "./componentRegistry";

interface Props {
  schema: UISchema;
}

export function ComponentRenderer({ schema }: Props) {
  return (
    <div className="space-y-4">
      {schema.components
        .filter((c) => c.visibility !== "hidden")
        .sort((a, b) => a.priority - b.priority)
        .map((c) => {
          const Component = componentRegistry[c.type];
          if (!Component) return null;
          return <Component key={`${c.type}-${c.priority}`} component={c} />;
        })}
    </div>
  );
}
