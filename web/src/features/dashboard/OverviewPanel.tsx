import { Button, toast } from "@heroui/react";
import {
  Activity,
  CircleAlert,
  Clock3,
  ListChecks,
  RefreshCw,
  Server,
} from "lucide-react";
import { useState } from "react";
import useSWR from "swr";

import type {
  Job,
  JobListResponse,
  NodeListResponse,
} from "../../api/types";
import { api, fetcher } from "../../lib/api";
import { cn } from "../../utils/classnames";
import { formatDate } from "../../utils/date";
import { errorMessage } from "../../utils/errors";
import { statusClass } from "../../utils/status";

export const OverviewPanel = () => {
  const [pending, setPending] = useState("");
  const { data: nodes, mutate: mutateNodes } =
    useSWR<NodeListResponse>("nodes", fetcher);
  const { data: jobs, mutate: mutateJobs } =
    useSWR<JobListResponse>("jobs", fetcher);
  const nodeItems = nodes?.items ?? [];
  const jobItems = jobs?.items ?? [];
  const online = nodeItems.filter((node) => node.status === "online").length;
  const running = jobItems.filter(
    (job) => job.status === "queued" || job.status === "running",
  ).length;
  const failed = jobItems.filter((job) => job.status === "failed").length;

  const probe = async (id: string) => {
    setPending(id);
    try {
      await api
        .post(`nodes/${encodeURIComponent(id)}/probe`)
        .json<Job>();
      await Promise.all([mutateNodes(), mutateJobs()]);
      toast.success("Agent check completed");
    } catch (error) {
      toast.danger("Agent check failed", {
        description: await errorMessage(error),
      });
    } finally {
      setPending("");
    }
  };

  const stats = [
    {
      label: "Server status",
      value: nodeItems.length && online === nodeItems.length ? "Healthy" : "Attention",
      note: `${online} of ${nodeItems.length} nodes online`,
      icon: Activity,
      tone: online === nodeItems.length ? "success" : "warning",
    },
    {
      label: "Managed nodes",
      value: String(nodeItems.length),
      note: "Local and remote agents",
      icon: Server,
      tone: "accent",
    },
    {
      label: "Active jobs",
      value: String(running),
      note: "Queued or running now",
      icon: ListChecks,
      tone: "accent",
    },
    {
      label: "Failed jobs",
      value: String(failed),
      note: "In recent activity",
      icon: CircleAlert,
      tone: failed ? "danger" : "default",
    },
  ];

  return (
    <div className="space-y-6">
      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {stats.map((stat) => {
          const Icon = stat.icon;
          return (
            <div key={stat.label} className="metric-card">
              <div className="flex items-start justify-between">
                <div className="text-sm text-foreground-500">{stat.label}</div>
                <div
                  className={cn(
                    "flex size-9 items-center justify-center rounded-xl",
                    stat.tone === "success"
                      ? "bg-success/12 text-success"
                      : stat.tone === "danger"
                        ? "bg-danger/12 text-danger"
                        : stat.tone === "warning"
                          ? "bg-warning/12 text-warning"
                          : "bg-accent/12 text-accent",
                  )}
                >
                  <Icon className="size-4" />
                </div>
              </div>
              <div className="mt-5 text-2xl font-semibold tracking-tight">
                {stat.value}
              </div>
              <div className="mt-1 text-xs text-foreground-400">
                {stat.note}
              </div>
            </div>
          );
        })}
      </section>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_24rem]">
        <section className="panel-card overflow-hidden">
          <div className="border-b border-divider px-6 py-5">
            <h2 className="text-base font-semibold">Managed nodes</h2>
            <div className="mt-1 text-sm text-foreground-500">
              Agent connectivity for this installation.
            </div>
          </div>
          <div className="divide-y divide-divider">
            {nodeItems.length ? (
              nodeItems.map((node) => (
                <div
                  key={node.id}
                  className="flex flex-col gap-4 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div className="flex items-center gap-4">
                    <div className="icon-box">
                      <Server className="size-5" />
                    </div>
                    <div>
                      <div className="flex items-center gap-2 font-medium">
                        {node.name}
                        <span
                          className={cn(
                            "rounded-full px-2 py-0.5 text-xs capitalize",
                            statusClass(node.status),
                          )}
                        >
                          {node.status}
                        </span>
                      </div>
                      <div className="mt-1 text-xs text-foreground-400">
                        {node.kind} · last seen {formatDate(node.lastSeenAt)}
                      </div>
                    </div>
                  </div>
                  <Button
                    size="sm"
                    variant="secondary"
                    isDisabled={pending === node.id}
                    onPress={() => void probe(node.id)}
                  >
                    <RefreshCw
                      className={cn(
                        "size-4",
                        pending === node.id && "animate-spin",
                      )}
                    />
                    Check agent
                  </Button>
                </div>
              ))
            ) : (
              <div className="px-6 py-12 text-center text-sm text-foreground-400">
                No nodes configured.
              </div>
            )}
          </div>
        </section>

        <section className="panel-card overflow-hidden">
          <div className="border-b border-divider px-6 py-5">
            <h2 className="text-base font-semibold">Recent activity</h2>
          </div>
          <div className="divide-y divide-divider">
            {jobItems.length ? (
              jobItems.slice(0, 6).map((job) => (
                <div key={job.id} className="flex gap-3 px-5 py-4">
                  <Clock3 className="mt-0.5 size-4 shrink-0 text-foreground-400" />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center justify-between gap-3">
                      <div className="truncate text-sm font-medium">
                        {job.kind}
                      </div>
                      <span
                        className={cn(
                          "shrink-0 rounded-full px-2 py-0.5 text-[10px] capitalize",
                          statusClass(job.status),
                        )}
                      >
                        {job.status}
                      </span>
                    </div>
                    <div className="mt-1 text-xs text-foreground-400">
                      {formatDate(job.createdAt)}
                    </div>
                  </div>
                </div>
              ))
            ) : (
              <div className="px-6 py-12 text-center text-sm text-foreground-400">
                No recent activity.
              </div>
            )}
          </div>
        </section>
      </div>
    </div>
  );
};
