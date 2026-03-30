import { useState, useMemo, useEffect, useRef } from "react";
import { Terminal, Search, Copy, Pause, Play, X } from "lucide-react";
import type { LogLine } from "@/lib/types";

type LogLevelFilter = "all" | "info" | "warn" | "error";

function levelTone(level: string) {
  switch (level) {
    case "error":
      return "text-red-400";
    case "warn":
      return "text-amber-300";
    default:
      return "text-slate-400";
  }
}

function levelBadgeTone(level: string) {
  switch (level) {
    case "error":
      return "border-red-500/25 bg-red-500/10 text-red-300";
    case "warn":
      return "border-amber-500/25 bg-amber-500/10 text-amber-300";
    default:
      return "border-white/10 bg-white/5 text-slate-300";
  }
}

function formatLogTimestamp(timestamp: string) {
  return new Date(timestamp).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  });
}

function tagTone(seed: string) {
  const tones = [
    "border-sky-500/20 bg-sky-500/10 text-sky-200",
    "border-emerald-500/20 bg-emerald-500/10 text-emerald-200",
    "border-violet-500/20 bg-violet-500/10 text-violet-200",
    "border-cyan-500/20 bg-cyan-500/10 text-cyan-200",
    "border-orange-500/20 bg-orange-500/10 text-orange-200",
    "border-fuchsia-500/20 bg-fuchsia-500/10 text-fuchsia-200",
  ];
  let hash = 0;
  for (let index = 0; index < seed.length; index += 1) {
    hash = (hash * 31 + seed.charCodeAt(index)) >>> 0;
  }
  return tones[hash % tones.length];
}

export function LogViewer({ logs, title = "Logs" }: { logs: LogLine[]; title?: string }) {
  const [search, setSearch] = useState("");
  const [containerFilter, setContainerFilter] = useState("all");
  const [levelFilter, setLevelFilter] = useState<LogLevelFilter>("all");
  const [autoScroll, setAutoScroll] = useState(true);
  const [copiedLogId, setCopiedLogId] = useState<string | null>(null);
  const [logsCleared, setLogsCleared] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const shouldStickToBottomRef = useRef(true);

  const containerOptions = useMemo(() => {
    return Array.from(new Set(logs.map((log) => log.containerName).filter((value) => value.trim() !== ""))).sort();
  }, [logs]);

  const filteredLogs = useMemo(() => {
    const lowerSearch = search.toLowerCase();
    return logs.filter((log) => {
      const matchesContainer = containerFilter === "all" || log.containerName === containerFilter;
      const matchesLevel = levelFilter === "all" || log.level === levelFilter;
      const matchesSearch =
        search.trim() === "" ||
        log.message.toLowerCase().includes(lowerSearch) ||
        log.containerName.toLowerCase().includes(lowerSearch) ||
        log.serviceTag.toLowerCase().includes(lowerSearch) ||
        log.level.toLowerCase().includes(lowerSearch);

      return matchesContainer && matchesLevel && matchesSearch;
    });
  }, [logs, search, containerFilter, levelFilter]);

  const visibleLogs = useMemo(() => (logsCleared ? [] : filteredLogs), [logsCleared, filteredLogs]);

  useEffect(() => {
    if (autoScroll && shouldStickToBottomRef.current && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [visibleLogs, autoScroll]);

  useEffect(() => {
    if (!copiedLogId) {
      return undefined;
    }
    const timeout = window.setTimeout(() => setCopiedLogId(null), 1400);
    return () => window.clearTimeout(timeout);
  }, [copiedLogId]);

  return (
    <div className="flex flex-col rounded-lg border border-border bg-card overflow-hidden">
      <div className="flex items-center justify-between border-b border-border bg-muted/40 px-4 py-3">
        <div className="flex items-center gap-2 font-medium text-foreground">
          <Terminal className="h-4 w-4" />
          {title} <span className="text-xs text-muted-foreground ml-2 font-normal">({visibleLogs.length} lines)</span>
        </div>
      </div>

      <div className="sticky top-0 z-10 border-b border-border bg-card/95 px-4 py-3 backdrop-blur">
        <div className="flex flex-wrap items-center gap-2">
          <select
            value={containerFilter}
            onChange={(event) => {
              setLogsCleared(false);
              setContainerFilter(event.target.value);
            }}
            className="h-8 rounded-md border border-border bg-background px-2.5 text-xs text-foreground outline-none focus:ring-1 focus:ring-ring"
          >
            <option value="all">All containers</option>
            {containerOptions.map((containerName) => (
              <option key={containerName} value={containerName}>
                {containerName}
              </option>
            ))}
          </select>
          <select
            value={levelFilter}
            onChange={(event) => {
              setLogsCleared(false);
              setLevelFilter(event.target.value as LogLevelFilter);
            }}
            className="h-8 rounded-md border border-border bg-background px-2.5 text-xs uppercase text-foreground outline-none focus:ring-1 focus:ring-ring"
          >
            <option value="all">All levels</option>
            <option value="info">INFO</option>
            <option value="warn">WARN</option>
            <option value="error">ERROR</option>
          </select>
          <div className="relative">
            <Search className="absolute left-2.5 top-2 h-3.5 w-3.5 text-muted-foreground" />
            <input
              type="text"
              placeholder="Search logs..."
              value={search}
              onChange={(e) => {
                setLogsCleared(false);
                setSearch(e.target.value);
              }}
              className="h-8 w-52 rounded-md border border-border bg-background pl-8 pr-2.5 text-xs text-foreground placeholder:text-muted-foreground outline-none focus:ring-1 focus:ring-ring"
            />
          </div>
          <button
            type="button"
            onClick={() => {
              const next = !autoScroll;
              setAutoScroll(next);
              shouldStickToBottomRef.current = next;
              if (next && scrollRef.current) {
                scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
              }
            }}
            className="inline-flex h-8 items-center gap-1.5 rounded-md border border-border bg-background px-2.5 text-xs text-muted-foreground transition-colors hover:text-foreground"
          >
            {autoScroll ? <Pause className="h-3.5 w-3.5" /> : <Play className="h-3.5 w-3.5" />}
            {autoScroll ? "Pause scroll" : "Resume scroll"}
          </button>
          <button
            type="button"
            onClick={() => {
              setLogsCleared(true);
              setAutoScroll(false);
            }}
            className="inline-flex h-8 items-center gap-1.5 rounded-md border border-border bg-background px-2.5 text-xs text-muted-foreground transition-colors hover:text-foreground"
          >
            <X className="h-3.5 w-3.5" />
            Clear logs
          </button>
        </div>
      </div>
      
      <div
        ref={scrollRef}
        className="h-96 overflow-y-auto bg-[#0A0A0B] px-4 py-3 text-[13px] font-mono leading-6"
        style={{ scrollBehavior: "auto" }}
        onScroll={(e) => {
          const target = e.target as HTMLDivElement;
          const isAtBottom = Math.abs(target.scrollHeight - target.clientHeight - target.scrollTop) < 10;
          shouldStickToBottomRef.current = isAtBottom;
          if (!isAtBottom && autoScroll) {
            setAutoScroll(false);
          }
        }}
      >
        {visibleLogs.length === 0 ? (
          <div className="text-muted-foreground/50 text-center py-10">No logs found</div>
        ) : (
          <div className="space-y-1.5">
            {visibleLogs.map((log) => (
              <div
                key={log.id}
                className="group rounded-md px-2 py-1.5 transition-colors hover:bg-white/5"
              >
                <div className="flex items-start gap-3">
                  <span className="shrink-0 text-xs text-muted-foreground/70 tabular-nums">
                    [{formatLogTimestamp(log.timestamp)}]
                  </span>
                  <span
                    className={`inline-flex shrink-0 items-center rounded-full border px-2 py-0.5 text-[11px] ${tagTone(log.containerName || log.serviceTag)}`}
                    title={log.containerName}
                  >
                    {log.serviceTag || log.containerName}
                  </span>
                  <span className={`inline-flex shrink-0 items-center rounded-full border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-[0.14em] ${levelBadgeTone(log.level)} ${levelTone(log.level)}`}>
                    {log.level}
                  </span>
                  <span className="min-w-0 flex-1 whitespace-pre-wrap break-words text-foreground/92">
                    {log.message}
                  </span>
                  <button
                    type="button"
                    onClick={() => {
                      void navigator.clipboard.writeText(
                        `[${formatLogTimestamp(log.timestamp)}] [${log.serviceTag || log.containerName}] ${log.level.toUpperCase()} ${log.message}`
                      );
                      setCopiedLogId(log.id);
                    }}
                    className="mt-0.5 hidden shrink-0 items-center gap-1 rounded-md border border-border bg-background/60 px-2 py-1 text-[11px] text-muted-foreground transition-colors hover:text-foreground group-hover:inline-flex"
                  >
                    <Copy className="h-3 w-3" />
                    {copiedLogId === log.id ? "Copied" : "Copy"}
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
