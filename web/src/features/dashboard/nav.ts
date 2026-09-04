import {
  Clock3,
  Database,
  Globe2,
  HardDrive,
  LayoutDashboard,
  ListTodo,
  LockKeyhole,
  Users,
} from "lucide-react";

export const views = [
  "overview",
  "accounts",
  "domains",
  "certificates",
  "databases",
  "cron",
  "backups",
  "jobs",
] as const;

export type View = (typeof views)[number];

export const nav = [
  {
    label: "Workspace",
    items: [{ id: "overview", label: "Overview", icon: LayoutDashboard }],
  },
  {
    label: "Hosting",
    items: [
      { id: "accounts", label: "Accounts", icon: Users },
      { id: "domains", label: "Domains", icon: Globe2 },
      { id: "certificates", label: "SSL / TLS", icon: LockKeyhole },
      { id: "databases", label: "Databases", icon: Database },
    ],
  },
  {
    label: "Automation",
    items: [
      { id: "cron", label: "Cron jobs", icon: Clock3 },
      { id: "backups", label: "Backups", icon: HardDrive },
    ],
  },
  {
    label: "System",
    items: [{ id: "jobs", label: "Jobs", icon: ListTodo }],
  },
] satisfies Array<{
  label: string;
  items: Array<{
    id: View;
    label: string;
    icon: typeof LayoutDashboard;
  }>;
}>;

export const page: Record<View, { title: string; description: string }> = {
  overview: {
    title: "Overview",
    description: "Server health and recent control-plane activity.",
  },
  accounts: {
    title: "Accounts",
    description: "Isolated hosting identities and system users.",
  },
  domains: {
    title: "Domains",
    description: "Nginx sites, aliases and document roots.",
  },
  certificates: {
    title: "SSL / TLS",
    description: "Certificates, renewals and HTTPS policy.",
  },
  databases: {
    title: "Databases",
    description: "MySQL databases, users and access grants.",
  },
  cron: {
    title: "Cron jobs",
    description: "Scheduled commands for hosting accounts.",
  },
  backups: {
    title: "Backups",
    description: "Plans, verified artifacts and restores.",
  },
  jobs: {
    title: "Jobs",
    description: "Durable operations executed by the local agent.",
  },
};
