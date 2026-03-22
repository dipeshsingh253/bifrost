import type { ReactNode } from "react";

type Props = {
  title: string;
  description?: string;
  cornerEl?: ReactNode;
  children: ReactNode;
};

export function ChartCard({ title, description, cornerEl, children }: Props) {
  return (
    <div className="rounded-lg border border-border bg-card overflow-hidden">
      <div className="flex items-start justify-between px-4 pt-4 pb-2">
        <div className="min-w-0">
          <h3 className="text-sm font-semibold text-foreground">{title}</h3>
          {description && (
            <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>
          )}
        </div>
        {cornerEl && <div className="ml-3 shrink-0">{cornerEl}</div>}
      </div>
      <div className="px-2 pb-3">{children}</div>
    </div>
  );
}
