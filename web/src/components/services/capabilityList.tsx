import type { ServiceCapabilities } from "@/contracts/types";
import { cn } from "@/utils/classnames";

type Props = {
    capabilities: ServiceCapabilities;
};

const groups = [
    ["Web server", "webservers"],
    ["Runtime", "runtimes"],
    ["Database", "databases"],
    ["Scheduler", "schedulers"],
    ["Backup storage", "backups"],
] as const;

const CapabilityList = ({ capabilities }: Props) => (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
        {groups.map(([label, key]) => {
            const items = capabilities[key];

            return (
                <div
                    key={key}
                    className="rounded-xl border border-divider bg-surface-secondary/45 p-4"
                >
                    <div className="text-xs font-medium text-foreground-400">{label}</div>
                    <div className="mt-3 space-y-2">
                        {items.map((item) => (
                            <div
                                key={`${item.driver}:${item.version}`}
                                className="flex items-center gap-2"
                            >
                                <span
                                    className={cn(
                                        "size-2 rounded-full",
                                        item.status === "healthy" ? "bg-success" : "bg-danger",
                                    )}
                                    aria-hidden="true"
                                />
                                <div className="min-w-0">
                                    <div className="truncate text-sm font-medium">
                                        {item.driver}
                                        {item.version ? ` ${item.version}` : ""}
                                    </div>
                                    <div
                                        className={cn(
                                            "text-xs capitalize",
                                            item.status === "healthy"
                                                ? "text-success"
                                                : "text-danger",
                                        )}
                                    >
                                        {item.status}
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            );
        })}
    </div>
);

export default CapabilityList;
