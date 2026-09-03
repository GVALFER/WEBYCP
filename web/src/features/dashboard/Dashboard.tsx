import { Button } from "@heroui/react";
import {
  Activity,
  Clock3,
  Database,
  Globe2,
  HardDrive,
  LockKeyhole,
  LogOut,
  RefreshCw,
  Server,
} from "lucide-react";
import { useState } from "react";
import useSWR from "swr";
import { useUrlState } from "urlstate-js";

import { AccountsPanel } from "../accounts/AccountsPanel";
import { BackupsPanel } from "../backups/BackupsPanel";
import { CertificatesPanel } from "../certificates/CertificatesPanel";
import { CronPanel } from "../cron/CronPanel";
import { DatabasesPanel } from "../databases/DatabasesPanel";
import { DomainsPanel } from "../domains/DomainsPanel";
import {
  fetcher,
  logout,
  probeNode,
  type AuthResponse,
  type JobListResponse,
  type NodeListResponse,
} from "../../lib/api";
import { cn } from "../../utils/classnames";
import { formatDate } from "../../utils/date";
import { errorMessage } from "../../utils/errors";
import { statusClass } from "../../utils/status";

const views = ["overview", "accounts", "domains", "certificates", "databases", "cron", "backups", "jobs"] as const;
const viewLabels: Record<(typeof views)[number], string> = {
  overview: "Overview",
  accounts: "Accounts",
  domains: "Domains",
  certificates: "SSL/TLS",
  databases: "Databases",
  cron: "Cron",
  backups: "Backups",
  jobs: "Jobs",
};

type Props = {
  session: AuthResponse;
  onLogout: () => void;
};

export const Dashboard = ({ session, onLogout }: Props) => {
  const [view, setView] = useUrlState("view", {
    default: "overview",
    values: views,
  });
  const [pendingNode, setPendingNode] = useState("");
  const [actionError, setActionError] = useState("");
  const { data: nodes, mutate: mutateNodes } = useSWR<NodeListResponse>(
    "nodes",
    fetcher,
    { refreshInterval: 4_000 },
  );
  const { data: jobs, mutate: mutateJobs } = useSWR<JobListResponse>(
    "jobs",
    fetcher,
    { refreshInterval: 2_000 },
  );

  const runProbe = async (nodeId: string) => {
    setPendingNode(nodeId);
    setActionError("");
    try {
      await probeNode(nodeId, session.csrfToken);
      await mutateJobs();
      setTimeout(() => void mutateNodes(), 500);
    } catch (error) {
      setActionError(await errorMessage(error));
    } finally {
      setPendingNode("");
    }
  };

  const signOut = async () => {
    try {
      await logout(session.csrfToken);
    } finally {
      onLogout();
    }
  };

  return (
    <div className="min-h-screen bg-background text-foreground">
      <header className="border-b border-divider bg-surface/70 backdrop-blur">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-6 py-4 lg:px-10">
          <div className="flex items-center gap-3 font-semibold">
            <div className="flex size-9 items-center justify-center rounded-xl bg-accent text-accent-foreground">
              <Server className="size-4" aria-hidden="true" />
            </div>
            WEBYCP
          </div>
          <div className="flex items-center gap-3">
            <div className="hidden text-right sm:block">
              <div className="text-sm font-medium">{session.user.name}</div>
              <div className="text-xs text-foreground-400">
                {session.user.email}
              </div>
            </div>
            <Button
              variant="tertiary"
              isIconOnly
              aria-label="Sign out"
              onPress={signOut}
            >
              <LogOut className="size-4" aria-hidden="true" />
            </Button>
          </div>
        </div>
      </header>

      <div className="mx-auto max-w-7xl px-6 py-8 lg:px-10">
        <div className="mb-8 flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <div className="text-sm font-medium tracking-[0.18em] text-accent uppercase">
              Control plane
            </div>
            <h1 className="mt-2 text-3xl font-semibold tracking-tight">
              {view === "overview" ? "Server overview" : viewLabels[view]}
            </h1>
          </div>
          <div className="flex flex-wrap gap-2">
            {views.map((item) => (
              <Button
                key={item}
                variant={view === item ? "primary" : "tertiary"}
                onPress={() => setView(item, { history: "push" })}
              >
                {viewLabels[item]}
              </Button>
            ))}
          </div>
        </div>

        {actionError && (
          <div className="mb-6 rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">
            {actionError}
          </div>
        )}

        {view === "overview" ? (
          <div className="space-y-8">
            <section className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              {[
                { icon: Globe2, label: "Web server", value: "Nginx" },
                { icon: Database, label: "Database", value: "MySQL 8.0" },
                { icon: LockKeyhole, label: "Certificates", value: "ACME" },
                { icon: HardDrive, label: "Backups", value: "Local" },
              ].map(({ icon: Icon, label, value }) => (
                <div
                  key={label}
                  className="rounded-2xl border border-divider bg-surface p-5 shadow-sm"
                >
                  <Icon
                    className="mb-5 size-5 text-accent"
                    aria-hidden="true"
                  />
                  <div className="text-sm text-foreground-500">{label}</div>
                  <div className="mt-1 text-lg font-medium">{value}</div>
                </div>
              ))}
            </section>

            <section className="rounded-2xl border border-divider bg-surface">
              <div className="border-b border-divider px-6 py-5">
                <h2 className="text-lg font-semibold">Managed nodes</h2>
                <div className="mt-1 text-sm text-foreground-500">
                  Agent connectivity for this installation.
                </div>
              </div>
              <div className="divide-y divide-divider">
                {nodes?.items.map((node) => (
                  <div
                    key={node.id}
                    className="flex flex-col gap-4 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"
                  >
                    <div className="flex items-center gap-4">
                      <div className="flex size-10 items-center justify-center rounded-xl bg-default/10">
                        <Activity className="size-5" aria-hidden="true" />
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
                          Last seen: {formatDate(node.lastSeenAt)}
                        </div>
                      </div>
                    </div>
                    <Button
                      variant="secondary"
                      isDisabled={pendingNode === node.id}
                      onPress={() => void runProbe(node.id)}
                    >
                      <RefreshCw
                        className={cn(
                          "size-4",
                          pendingNode === node.id && "animate-spin",
                        )}
                        aria-hidden="true"
                      />
                      Check agent
                    </Button>
                  </div>
                ))}
              </div>
            </section>
          </div>
        ) : view === "accounts" ? (
          <AccountsPanel
            nodeId={nodes?.items[0]?.id ?? ""}
            session={session}
          />
        ) : view === "domains" ? (
          <DomainsPanel session={session} />
        ) : view === "certificates" ? (
          <CertificatesPanel session={session} />
        ) : view === "databases" ? (
          <DatabasesPanel session={session} />
        ) : view === "cron" ? (
          <CronPanel session={session} />
        ) : view === "backups" ? (
          <BackupsPanel session={session} />
        ) : (
          <section className="overflow-hidden rounded-2xl border border-divider bg-surface">
            <div className="border-b border-divider px-6 py-5">
              <h2 className="text-lg font-semibold">Recent jobs</h2>
              <div className="mt-1 text-sm text-foreground-500">
                Durable operations executed by the local agent.
              </div>
            </div>
            <div className="divide-y divide-divider">
              {jobs?.items.length ? (
                jobs.items.map((job) => (
                  <div
                    key={job.id}
                    className="grid gap-3 px-6 py-5 sm:grid-cols-[1fr_auto_auto] sm:items-center sm:gap-8"
                  >
                    <div className="flex items-center gap-3">
                      <Clock3
                        className="size-4 text-foreground-400"
                        aria-hidden="true"
                      />
                      <div>
                        <div className="font-medium">{job.kind}</div>
                        <div className="text-xs text-foreground-400">
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
        )}
      </div>
    </div>
  );
};
