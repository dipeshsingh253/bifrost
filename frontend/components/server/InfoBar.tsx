import type { Server } from "@/lib/types";
import { Globe, Clock, Cpu, HardDrive } from "lucide-react";

type Props = {
  server: Server;
};

export function InfoBar({ server }: Props) {
  const uptimeDays = Math.floor(server.uptime_seconds / 86400);

  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5 text-sm text-muted-foreground">
      <span className="flex items-center gap-1.5">
        <span
          className={`h-2.5 w-2.5 rounded-full ${
            server.status === "up" ? "bg-success" : "bg-destructive"
          }`}
        />
        <span className="font-medium text-foreground capitalize">{server.status}</span>
      </span>

      <span className="text-border">|</span>

      <span className="flex items-center gap-1.5">
        <Globe className="h-3.5 w-3.5" />
        {server.hostname}
      </span>

      <span className="text-border">|</span>

      <span className="flex items-center gap-1.5">
        <HardDrive className="h-3.5 w-3.5" />
        {server.os}
      </span>

      <span className="text-border">|</span>

      <span className="flex items-center gap-1.5">
        <Clock className="h-3.5 w-3.5" />
        {uptimeDays} days
      </span>

      <span className="text-border">|</span>

      <span>{server.kernel}</span>

      <span className="text-border">|</span>

      <span className="flex items-center gap-1.5">
        <Cpu className="h-3.5 w-3.5" />
        {server.cpu_model} ({server.cpu_cores}c/{server.cpu_threads}t)
      </span>
    </div>
  );
}
