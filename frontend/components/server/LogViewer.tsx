import { useState, useMemo, useEffect, useRef } from "react";
import { Terminal, Search } from "lucide-react";
import type { LogLine } from "@/lib/types";

export function LogViewer({ logs, title = "Logs" }: { logs: LogLine[]; title?: string }) {
  const [search, setSearch] = useState("");
  const [autoScroll, setAutoScroll] = useState(true);
  const scrollRef = useRef<HTMLDivElement>(null);

  const filteredLogs = useMemo(() => {
    if (!search) return logs;
    const lowerSearch = search.toLowerCase();
    return logs.filter(
      (l) =>
        l.message.toLowerCase().includes(lowerSearch) ||
        l.containerName.toLowerCase().includes(lowerSearch) ||
        l.serviceTag.toLowerCase().includes(lowerSearch)
    );
  }, [logs, search]);

  useEffect(() => {
    if (autoScroll && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [filteredLogs, autoScroll]);

  return (
    <div className="flex flex-col rounded-lg border border-border bg-card overflow-hidden">
      <div className="flex items-center justify-between border-b border-border bg-muted/40 px-4 py-3">
        <div className="flex items-center gap-2 font-medium text-foreground">
          <Terminal className="h-4 w-4" />
          {title} <span className="text-xs text-muted-foreground ml-2 font-normal">({filteredLogs.length} lines)</span>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <div className="relative">
            <Search className="absolute left-2.5 top-1.5 h-3.5 w-3.5 text-muted-foreground" />
            <input
              type="text"
              placeholder="Search logs..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="h-7 w-48 rounded-md border border-border bg-background pl-8 pr-2.5 text-xs text-foreground placeholder:text-muted-foreground outline-none focus:ring-1 focus:ring-ring"
            />
          </div>
          <label className="flex items-center gap-2 text-xs text-muted-foreground cursor-pointer select-none">
            <input
              type="checkbox"
              checked={autoScroll}
              onChange={(e) => setAutoScroll(e.target.checked)}
              className="rounded border-border bg-background text-primary focus:ring-primary"
            />
            Auto-scroll
          </label>
        </div>
      </div>
      
      <div
        ref={scrollRef}
        className="h-96 overflow-y-auto bg-[#0A0A0B] p-4 text-[13px] font-mono leading-relaxed"
        style={{ scrollBehavior: "auto" }}
        onScroll={(e) => {
          const target = e.target as HTMLDivElement;
          const isAtBottom = Math.abs(target.scrollHeight - target.clientHeight - target.scrollTop) < 10;
          if (!isAtBottom && autoScroll) setAutoScroll(false);
        }}
      >
        {filteredLogs.length === 0 ? (
          <div className="text-muted-foreground/50 text-center py-10">No logs found</div>
        ) : (
          filteredLogs.map((log) => (
            <div key={log.id} className="flex flex-col sm:flex-row sm:gap-4 hover:bg-white/5 px-2 py-0.5 rounded -mx-2 transition-colors">
              <span className="text-muted-foreground/70 shrink-0 sm:w-44 tabular-nums">
                {new Date(log.timestamp).toISOString().replace('T', ' ').replace('Z', '')}
              </span>
              <div className="flex gap-4 sm:contents">
                <span className="text-[hsl(280_65%_70%)] shrink-0 sm:w-24 truncate" title={log.serviceTag}>
                  [{log.serviceTag}]
                </span>
                <span className="text-[hsl(160_60%_55%)] shrink-0 sm:w-32 truncate" title={log.containerName}>
                  {log.containerName}
                </span>
              </div>
              <span className="text-foreground/90 whitespace-pre-wrap break-all">{log.message}</span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
