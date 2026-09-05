"use client";

import { Clock3 } from "lucide-react";
import useSWR from "swr";
import { Table, type TableColumn } from "@/components/table/table";
import { useTable } from "@/components/table/useTable";
import type {
    AccountListResponse,
    ScheduledTaskListResponse,
    ServiceSettings,
} from "@/contracts/types";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";
import TaskForm from "./actions/taskForm";
import TaskActions from "./actions/taskActions";

type Props = {
    accounts: AccountListResponse;
    tasks: ScheduledTaskListResponse;
    settings: ServiceSettings;
};

type ScheduledTask = ScheduledTaskListResponse["items"][number];

const Tasks = ({ accounts, tasks, settings: initialSettings }: Props) => {
    const table = useTable(tasks.pagination);

    const { data, isLoading } = useSWR<ScheduledTaskListResponse>(`scheduled-tasks${table.query}`, {
        fallbackData: table.isInitialQuery ? tasks : undefined,
    });
    const { data: settings = initialSettings } = useSWR<ServiceSettings>("service-settings", {
        fallbackData: initialSettings,
    });

    const columns: TableColumn<ScheduledTask>[] = [
        {
            id: "task",
            label: "Task",
            isRowHeader: true,
            render: (task) => (
                <div className="flex min-w-0 items-center gap-4">
                    <div className="icon-box">
                        <Clock3 className="size-5" aria-hidden="true" />
                    </div>
                    <div className="min-w-0">
                        <div className="font-medium">{task.name}</div>
                        <div className="mt-1 max-w-xl truncate font-mono text-xs text-foreground-400">
                            {task.command} · {task.schedulerDriver}
                        </div>
                    </div>
                </div>
            ),
        },
        {
            id: "schedule",
            label: "Schedule (UTC)",
            cellClassName: "whitespace-nowrap font-mono text-xs text-foreground-500",
            render: (task) => task.schedule,
        },
        {
            id: "status",
            label: "Status",
            render: (task) => (
                <span
                    className={cn(
                        "rounded-full px-2.5 py-1 text-xs capitalize",
                        statusClass(task.status),
                    )}
                >
                    {task.status}
                </span>
            ),
        },
        {
            id: "actions",
            label: "Actions",
            headerClassName: "text-end",
            cellClassName: "w-px whitespace-nowrap",
            render: (task) => (
                <div className="flex items-center gap-2">
                    <TaskForm accounts={accounts} driver={task.schedulerDriver} task={task} />
                    <TaskActions item={task} />
                </div>
            ),
        },
    ];

    return (
        <section className="panel-card overflow-hidden">
            <div className="flex items-start justify-between gap-4 px-6 py-5">
                <div>
                    <h2 className="text-base font-semibold">Scheduled tasks</h2>
                    <div className="mt-1 text-sm text-foreground-500">
                        Commands run as the hosting account from its home directory.
                    </div>
                </div>
                <TaskForm accounts={accounts} driver={settings.defaults.schedulerDriver} />
            </div>
            <Table table={table} columns={columns} data={data} pending={isLoading} />
        </section>
    );
};

export default Tasks;
