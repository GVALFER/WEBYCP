export const statusClass = (status: string) =>
    status === "online" || status === "active" || status === "succeeded"
        ? "bg-success/15 text-success"
        : status === "offline" || status === "error" || status === "failed"
          ? "bg-danger/15 text-danger"
          : "bg-warning/15 text-warning";
