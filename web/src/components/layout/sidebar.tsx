"use client";

import { Button } from "@heroui/react";
import {
    Clock3,
    Database,
    Globe2,
    HardDrive,
    LayoutDashboard,
    ListTodo,
    LockKeyhole,
    Moon,
    Server,
    Sun,
    UserRoundCog,
    Users,
} from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useTheme } from "@/lib/theme";
import { useSession } from "@/providers/SessionProvider";
import { cn } from "@/utils/classnames";
import Logout from "../actions/logout";

export const pages = [
    {
        href: "/",
        label: "Overview",
        title: "Overview",
        description: "Server health and recent control-plane activity.",
        icon: LayoutDashboard,
        group: "Workspace",
    },
    {
        href: "/accounts",
        label: "Accounts",
        title: "Accounts",
        description: "Isolated hosting identities and system users.",
        icon: Users,
        group: "Hosting",
    },
    {
        href: "/domains",
        label: "Domains",
        title: "Domains",
        description: "Nginx sites, aliases and document roots.",
        icon: Globe2,
        group: "Hosting",
    },
    {
        href: "/certificates",
        label: "SSL / TLS",
        title: "SSL / TLS",
        description: "Certificates, renewals and HTTPS policy.",
        icon: LockKeyhole,
        group: "Hosting",
    },
    {
        href: "/databases",
        label: "Databases",
        title: "Databases",
        description: "MySQL databases, users and access grants.",
        icon: Database,
        group: "Hosting",
    },
    {
        href: "/cron",
        label: "Cron jobs",
        title: "Cron jobs",
        description: "Scheduled commands for hosting accounts.",
        icon: Clock3,
        group: "Automation",
    },
    {
        href: "/backups",
        label: "Backups",
        title: "Backups",
        description: "Plans, verified artifacts and restores.",
        icon: HardDrive,
        group: "Automation",
    },
    {
        href: "/jobs",
        label: "Jobs",
        title: "Jobs",
        description: "Durable operations executed by the local agent.",
        icon: ListTodo,
        group: "System",
    },
    {
        href: "/profile",
        label: "Profile",
        title: "Profile",
        description: "Administrator identity and sign-in credentials.",
        icon: UserRoundCog,
        group: "System",
    },
] as const;

const groups = ["Workspace", "Hosting", "Automation", "System"] as const;

export const getPage = (pathname: string) =>
    pages.find((item) =>
        item.href === "/"
            ? pathname === item.href
            : pathname === item.href || pathname.startsWith(`${item.href}/`),
    ) ?? pages[0];

type SidebarProps = {
    onNavigate?: () => void;
};

export const Sidebar = ({ onNavigate }: SidebarProps) => {
    const session = useSession();
    const pathname = usePathname();
    const { theme, toggleTheme } = useTheme();
    const ThemeIcon = theme === "dark" ? Sun : Moon;
    const current = getPage(pathname);

    return (
        <div className="flex h-full min-h-0 flex-col bg-surface p-4 text-surface-foreground">
            <div className="flex h-14 items-center gap-3 px-2">
                <div className="flex size-9 items-center justify-center rounded-xl bg-accent text-accent-foreground shadow-lg shadow-accent/20">
                    <Server className="size-4" aria-hidden="true" />
                </div>
                <div>
                    <div className="font-semibold tracking-tight">WEBYCP</div>
                    <div className="text-[10px] font-medium tracking-[0.16em] text-foreground-400 uppercase">
                        Control panel
                    </div>
                </div>
            </div>

            <nav
                className="mt-6 flex-1 space-y-6 overflow-y-auto px-1"
                aria-label="Main navigation"
            >
                {groups.map((group) => (
                    <div key={group}>
                        <div className="mb-2 px-3 text-[10px] font-semibold tracking-[0.16em] text-foreground-400 uppercase">
                            {group}
                        </div>
                        <div className="space-y-1">
                            {pages
                                .filter((item) => item.group === group)
                                .map((item) => {
                                    const Icon = item.icon;
                                    const active = item.href === current.href;

                                    return (
                                        <Link
                                            key={item.href}
                                            href={item.href}
                                            prefetch={false}
                                            aria-current={active ? "page" : undefined}
                                            className={cn(
                                                "group flex h-10 items-center gap-3 rounded-xl px-3 text-sm font-medium transition",
                                                active
                                                    ? "bg-accent/12 text-accent"
                                                    : "text-foreground-500 hover:bg-surface-secondary hover:text-foreground",
                                            )}
                                            onClick={onNavigate}
                                        >
                                            <Icon
                                                className={cn(
                                                    "size-4 transition",
                                                    active
                                                        ? "text-accent"
                                                        : "text-foreground-400 group-hover:text-foreground",
                                                )}
                                                aria-hidden="true"
                                            />
                                            {item.label}
                                        </Link>
                                    );
                                })}
                        </div>
                    </div>
                ))}
            </nav>

            <div className="mt-4 rounded-2xl border border-divider bg-surface-secondary/60 p-3">
                <div className="flex min-w-0 items-center gap-3">
                    <div className="flex size-9 shrink-0 items-center justify-center rounded-full bg-accent/15 text-sm font-semibold text-accent">
                        {session.user.name.charAt(0).toUpperCase()}
                    </div>
                    <div className="min-w-0 flex-1">
                        <div className="truncate text-sm font-medium">{session.user.name}</div>
                        <div className="truncate text-xs text-foreground-400">
                            @{session.user.username}
                        </div>
                    </div>
                </div>
                <div className="mt-3 flex gap-2 border-t border-divider pt-3">
                    <Button
                        fullWidth
                        size="sm"
                        variant="tertiary"
                        aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} theme`}
                        onPress={toggleTheme}
                    >
                        <ThemeIcon className="size-4" />
                        {theme === "dark" ? "Light theme" : "Dark theme"}
                    </Button>
                    <Logout />
                </div>
            </div>
        </div>
    );
};
