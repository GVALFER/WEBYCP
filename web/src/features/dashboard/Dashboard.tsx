import useSWR from "swr";
import { useUrlState } from "urlstate-js";

import type { AuthResponse, NodeListResponse } from "../../api/types";
import { api, fetcher } from "../../lib/api";
import { AccountsPanel } from "../accounts/AccountsPanel";
import { BackupsPanel } from "../backups/BackupsPanel";
import { CertificatesPanel } from "../certificates/CertificatesPanel";
import { CronPanel } from "../cron/CronPanel";
import { DatabasesPanel } from "../databases/DatabasesPanel";
import { DomainsPanel } from "../domains/DomainsPanel";
import { AppShell } from "./AppShell";
import { JobsPanel } from "./JobsPanel";
import { views, type View } from "./nav";
import { OverviewPanel } from "./OverviewPanel";

type Props = {
  session: AuthResponse;
  onLogout: () => void;
};

const AccountsView = () => {
  const { data } = useSWR<NodeListResponse>("nodes", fetcher);

  return <AccountsPanel nodeId={data?.items[0]?.id ?? ""} />;
};

export const Dashboard = ({ session, onLogout }: Props) => {
  const [view, setView] = useUrlState("view", {
    default: "overview",
    values: views,
  });

  const logout = async () => {
    try {
      await api.post("auth/logout");
    } finally {
      onLogout();
    }
  };

  const content =
    view === "overview" ? (
      <OverviewPanel />
    ) : view === "accounts" ? (
      <AccountsView />
    ) : view === "domains" ? (
      <DomainsPanel />
    ) : view === "certificates" ? (
      <CertificatesPanel session={session} />
    ) : view === "databases" ? (
      <DatabasesPanel />
    ) : view === "cron" ? (
      <CronPanel />
    ) : view === "backups" ? (
      <BackupsPanel />
    ) : (
      <JobsPanel />
    );

  return (
    <AppShell
      session={session}
      view={view}
      onView={(next: View) => setView(next, { history: "push" })}
      onLogout={() => void logout()}
    >
      {content}
    </AppShell>
  );
};
