import { AlertTriangle, Info, Play, Square, Replace } from "lucide-react";
import type { EventLog } from "@/lib/types";

export function EventsList({ events }: { events: EventLog[] }) {
  if (events.length === 0) {
    return (
      <div className="rounded-lg border border-border bg-card p-6 text-center text-muted-foreground">
        No recent events
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {events.map((event) => {
        let Icon = Info;
        let colorClass = "text-muted-foreground";
        let bgClass = "bg-muted/30";

        switch (event.type) {
          case "crash":
            Icon = AlertTriangle;
            colorClass = "text-destructive";
            bgClass = "bg-destructive/10";
            break;
          case "restart":
            Icon = Replace;
            colorClass = "text-orange-500";
            bgClass = "bg-orange-500/10";
            break;
          case "health_change":
            Icon = Info;
            colorClass = "text-blue-500";
            bgClass = "bg-blue-500/10";
            break;
          case "start":
            Icon = Play;
            colorClass = "text-success";
            bgClass = "bg-success/10";
            break;
          case "stop":
            Icon = Square;
            colorClass = "text-muted-foreground";
            bgClass = "bg-muted/10";
            break;
        }

        return (
          <div
            key={event.id}
            className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 rounded-md border border-border bg-card px-4 py-3"
          >
            <div className="flex items-center gap-3">
              <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full ${bgClass}`}>
                <Icon className={`h-4 w-4 ${colorClass}`} />
              </div>
              <div className="flex flex-col">
                <span className="font-medium text-foreground text-sm">
                  {event.entityName}
                </span>
                <span className="text-xs text-muted-foreground">
                  {event.message}
                </span>
              </div>
            </div>
            <div className="text-xs text-muted-foreground tabular-nums whitespace-nowrap">
              {new Date(event.timestamp).toLocaleString(undefined, {
                month: "short",
                day: "numeric",
                hour: "2-digit",
                minute: "2-digit",
                second: "2-digit",
              })}
            </div>
          </div>
        );
      })}
    </div>
  );
}
