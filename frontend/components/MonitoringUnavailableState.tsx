type MonitoringUnavailableStateProps = {
  message: string;
  title?: string;
};

export function MonitoringUnavailableState({
  message,
  title = "Monitoring data is temporarily unavailable.",
}: MonitoringUnavailableStateProps) {
  return (
    <div className="flex justify-center py-24">
      <div className="w-full max-w-2xl rounded-lg border border-destructive/30 bg-destructive/10 px-6 py-5 text-center">
        <h2 className="text-lg font-semibold text-foreground">{title}</h2>
        <p className="mt-2 text-sm text-muted-foreground">{message}</p>
      </div>
    </div>
  );
}
