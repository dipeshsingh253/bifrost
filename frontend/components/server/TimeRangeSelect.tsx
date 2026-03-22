import { Clock, LayoutGrid } from "lucide-react";

type Props = {
  selectedTime: string;
  onTimeChange: (time: string) => void;
  grid: boolean;
  onGridToggle: () => void;
};

const TIME_OPTIONS = [
  { label: "1 hour", value: "1h" },
  { label: "6 hours", value: "6h" },
  { label: "12 hours", value: "12h" },
  { label: "24 hours", value: "24h" },
  { label: "7 days", value: "7d" },
];

export function TimeRangeSelect({ selectedTime, onTimeChange, grid, onGridToggle }: Props) {
  return (
    <div className="flex items-center gap-2">
      <div className="flex items-center gap-1.5 rounded-md border border-border bg-background px-2 py-1">
        <Clock className="h-3.5 w-3.5 text-muted-foreground" />
        <select
          value={selectedTime}
          onChange={(e) => onTimeChange(e.target.value)}
          className="bg-transparent text-sm text-foreground outline-none cursor-pointer"
        >
          {TIME_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value} className="bg-card text-foreground">
              {opt.label}
            </option>
          ))}
        </select>
      </div>

      <button
        onClick={onGridToggle}
        className={`rounded-md border p-1.5 transition-colors ${
          grid
            ? "border-border bg-accent text-foreground"
            : "border-border bg-background text-muted-foreground hover:text-foreground"
        }`}
        title={grid ? "Switch to single column" : "Switch to grid"}
      >
        <LayoutGrid className="h-4 w-4" />
      </button>
    </div>
  );
}
