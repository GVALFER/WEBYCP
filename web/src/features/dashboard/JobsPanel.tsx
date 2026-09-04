import { Clock3 } from "lucide-react";
import useSWR from "swr";

import type { JobListResponse } from "../../api/types";
import { fetcher } from "../../lib/api";
import { cn } from "../../utils/classnames";
import { formatDate } from "../../utils/date";
import { statusClass } from "../../utils/status";

export const JobsPanel = () => {
  const { data } = useSWR<JobListResponse>("jobs", fetcher);

  return (
    <section className="panel-card overflow-hidden">
      <div className="border-b border-divider px-6 py-5">
        <h2 className="text-base font-semibold">Recent jobs</h2>
        <div className="mt-1 text-sm text-foreground-500">
          Durable operations executed by the local agent.
        </div>
      </div>
      <div className="divide-y divide-divider">
        {data?.items.length ? (
          data.items.map((job) => (
            <div
              key={job.id}
              className="grid gap-3 px-6 py-5 sm:grid-cols-[minmax(0,1fr)_auto_auto] sm:items-center sm:gap-8"
            >
              <div className="flex min-w-0 items-center gap-3">
                <div className="icon-box size-9">
                  <Clock3 className="size-4" />
                </div>
                <div className="min-w-0">
                  <div className="truncate font-medium">{job.kind}</div>
                  <div className="truncate font-mono text-xs text-foreground-400">
                    {job.id}
                  </div>
                </div>
              </div>
              <div className="text-sm text-foreground-500">
                {formatDate(job.createdAt)}
              </div>
              <span
                className={cn(
                  "w-fit rounded-full px-2.5 py-1 text-xs capitalize",
                  statusClass(job.status),
                )}
              >
                {job.status}
              </span>
            </div>
          ))
        ) : (
          <div className="px-6 py-12 text-center text-sm text-foreground-400">
            No jobs have run yet.
          </div>
        )}
      </div>
    </section>
  );
};
