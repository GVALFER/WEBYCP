"use client";

import type { LucideIcon } from "lucide-react";
import {
    Archive,
    ArchiveRestore,
    CalendarClock,
    ChevronDown,
    CircleUserRound,
    Code2,
    Clock3,
    Database,
    Dna,
    Globe2,
    HardDrive,
    LayoutDashboard,
    ListTodo,
    Link2,
    LockKeyhole,
    PackageOpen,
    Server,
    Settings2,
    Wrench,
    Users,
    Waypoints,
} from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import { Brand } from "@/components/brand";
import { useSession } from "@/providers/SessionProvider";
import { cn } from "@/utils/classnames";
import Logout from "../actions/logout";

type Page = {
    description: string;
    href: string;
    icon: LucideIcon;
    label: string;
    title: string;
};

type Group = {
    children: Page[];
    icon: LucideIcon;
    label: string;
};

const overview: Page = {
    href: "/",
    label: "Overview",
    title: "Overview",
    description: "Server health and recent control-plane activity.",
    icon: LayoutDashboard,
};

const groups: Group[] = [
    {
        label: "Accounts",
        icon: Users,
        children: [
            {
                href: "/accounts",
                label: "Accounts",
                title: "Accounts",
                description: "Isolated hosting identities and system users.",
                icon: CircleUserRound,
            },
            {
                href: "/packages",
                label: "Packages",
                title: "Packages",
                description: "Resource limits assigned to hosting accounts.",
                icon: PackageOpen,
            },
        ],
    },
    {
        label: "Websites",
        icon: Globe2,
        children: [
            {
                href: "/websites",
                label: "Websites",
                title: "Websites",
                description: "Sites, document roots and runtime stacks.",
                icon: Code2,
            },
            {
                href: "/domains",
                label: "Domains",
                title: "Domains",
                description: "Primary hostnames assigned to websites.",
                icon: Globe2,
            },
            {
                href: "/aliases",
                label: "Aliases",
                title: "Aliases",
                description: "Additional hostnames assigned to websites.",
                icon: Link2,
            },
            {
                href: "/certificates",
                label: "Certificates",
                title: "SSL / TLS",
                description: "Certificates, renewals and HTTPS policy.",
                icon: LockKeyhole,
            },
        ],
    },
    {
        label: "DNS",
        icon: Waypoints,
        children: [
            {
                href: "/dns/zones",
                label: "Zones",
                title: "DNS Zones",
                description: "Authoritative DNS zones managed by WEBYCP.",
                icon: Globe2,
            },
            {
                href: "/dns/providers",
                label: "Providers",
                title: "DNS Providers",
                description: "Authoritative DNS drivers and observed health.",
                icon: Dna,
            },
            {
                href: "/dns/nameservers",
                label: "Nameservers",
                title: "Nameservers",
                description: "Default authoritative nameservers and TTL.",
                icon: Server,
            },
        ],
    },
    {
        label: "Databases",
        icon: Database,
        children: [
            {
                href: "/databases",
                label: "Databases",
                title: "Databases",
                description: "MySQL databases, users and access grants.",
                icon: Database,
            },
        ],
    },
    {
        label: "Backups",
        icon: HardDrive,
        children: [
            {
                href: "/backups/plans",
                label: "Plans",
                title: "Backup Plans",
                description: "Schedules, retention and account backup runs.",
                icon: CalendarClock,
            },
            {
                href: "/backups/archives",
                label: "Archives",
                title: "Backup Archives",
                description: "Completed account backups and their contents.",
                icon: Archive,
            },
            {
                href: "/backups/restore",
                label: "Restore",
                title: "Restore",
                description: "Verified, selective account restores.",
                icon: ArchiveRestore,
            },
            {
                href: "/backups/destinations",
                label: "Destinations",
                title: "Backup Destinations",
                description: "Backup storage available on each server.",
                icon: HardDrive,
            },
        ],
    },
    {
        label: "Scheduled Tasks",
        icon: CalendarClock,
        children: [
            {
                href: "/cron",
                label: "Tasks",
                title: "Scheduled Tasks",
                description: "Scheduled commands for hosting accounts.",
                icon: Clock3,
            },
        ],
    },
    {
        label: "System",
        icon: Settings2,
        children: [
            {
                href: "/servers",
                label: "Servers",
                title: "Servers",
                description: "Managed nodes, Agent connectivity and installed capabilities.",
                icon: Server,
            },
            {
                href: "/services",
                label: "Services",
                title: "Services",
                description: "Observed services and defaults for new resources.",
                icon: Wrench,
            },
            {
                href: "/jobs",
                label: "Jobs",
                title: "Jobs",
                description: "Durable operations executed by the local agent.",
                icon: ListTodo,
            },
            {
                href: "/profile",
                label: "Profile",
                title: "Profile",
                description: "Administrator identity and sign-in credentials.",
                icon: CircleUserRound,
            },
            {
                href: "/settings",
                label: "Settings",
                title: "Settings",
                description: "Panel-wide services and configuration.",
                icon: Settings2,
            },
        ],
    },
];

export const pages = [overview, ...groups.flatMap((group) => group.children)];

const isActive = (pathname: string, href: string) =>
    href === "/" ? pathname === href : pathname === href || pathname.startsWith(`${href}/`);

export const getPage = (pathname: string) =>
    pages.find((page) => isActive(pathname, page.href)) ?? overview;

type SidebarProps = {
    onNavigate?: () => void;
};

export const Sidebar = ({ onNavigate }: SidebarProps) => {
    const session = useSession();
    const pathname = usePathname();

    return (
        <div className="sidebar-shell flex h-full min-h-0 flex-col px-3 py-4 text-surface-foreground">
            <Brand className="h-13 px-2" />

            <nav className="mt-5 flex-1 overflow-y-auto px-1" aria-label="Main navigation">
                <NavLink page={overview} active={pathname === "/"} onNavigate={onNavigate} />

                <div className="mt-5 space-y-1.5">
                    {groups.map((group) => (
                        <NavGroup
                            key={group.label}
                            group={group}
                            pathname={pathname}
                            onNavigate={onNavigate}
                        />
                    ))}
                </div>
            </nav>

            <div className="mt-4 border-t border-divider px-1 pt-4">
                <div className="flex items-center gap-3 rounded-xl px-2 py-2">
                    <Link
                        href="/profile"
                        prefetch={false}
                        className="flex min-w-0 flex-1 items-center gap-3 rounded-lg outline-none focus-visible:ring-2 focus-visible:ring-focus"
                        onClick={onNavigate}
                    >
                        <div className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-accent/12 text-sm font-semibold text-accent">
                            {session.user.name.charAt(0).toUpperCase()}
                        </div>
                        <div className="min-w-0 flex-1">
                            <div className="truncate text-sm font-medium">{session.user.name}</div>
                            <div className="truncate text-[11px] text-foreground-400">
                                @{session.user.username}
                            </div>
                        </div>
                    </Link>
                    <Logout />
                </div>
            </div>
        </div>
    );
};

type NavGroupProps = {
    group: Group;
    pathname: string;
    onNavigate?: () => void;
};

const NavGroup = ({ group, pathname, onNavigate }: NavGroupProps) => {
    const active = group.children.some((page) => isActive(pathname, page.href));
    const [expanded, setExpanded] = useState(false);
    const open = active || expanded;
    const Icon = group.icon;

    return (
        <div>
            <button
                className={cn(
                    "flex h-11 w-full items-center gap-3 rounded-xl px-3 text-sm font-medium outline-none transition-colors focus-visible:ring-2 focus-visible:ring-focus",
                    active
                        ? "text-foreground"
                        : "text-foreground-500 hover:bg-surface-secondary/70 hover:text-foreground",
                )}
                type="button"
                aria-expanded={open}
                onClick={() => {
                    if (!active) setExpanded((current) => !current);
                }}
            >
                <Icon
                    className={cn("size-[1.1rem]", active ? "text-accent" : "text-foreground-400")}
                    strokeWidth={1.8}
                    aria-hidden="true"
                />
                <span className="flex-1 text-start">{group.label}</span>
                <ChevronDown
                    className={cn(
                        "size-4 text-foreground-400 transition-transform",
                        open && "rotate-180",
                    )}
                    aria-hidden="true"
                />
            </button>

            {open && (
                <div className="relative ml-[1.3rem] border-l border-divider py-1 pl-[1.15rem]">
                    {group.children.map((page) => (
                        <NavLink
                            key={page.href}
                            page={page}
                            active={isActive(pathname, page.href)}
                            nested
                            onNavigate={onNavigate}
                        />
                    ))}
                </div>
            )}
        </div>
    );
};

type NavLinkProps = {
    active: boolean;
    nested?: boolean;
    page: Page;
    onNavigate?: () => void;
};

const NavLink = ({ active, nested = false, page, onNavigate }: NavLinkProps) => {
    const Icon = page.icon;

    return (
        <Link
            href={page.href}
            prefetch={false}
            aria-current={active ? "page" : undefined}
            className={cn(
                "group relative flex h-10 items-center gap-3 rounded-xl px-3 text-sm font-medium outline-none transition-colors focus-visible:ring-2 focus-visible:ring-focus",
                active
                    ? "bg-accent/10 text-accent"
                    : "text-foreground-500 hover:bg-surface-secondary/70 hover:text-foreground",
            )}
            onClick={onNavigate}
        >
            {nested && (
                <span
                    className={cn(
                        "absolute top-1/2 left-[-1.42rem] size-2 -translate-y-1/2 rounded-full border-2 border-surface",
                        active ? "bg-accent" : "bg-border",
                    )}
                    aria-hidden="true"
                />
            )}
            <Icon
                className={cn("size-4", active ? "text-accent" : "text-foreground-400")}
                strokeWidth={1.8}
                aria-hidden="true"
            />
            <span>{page.label}</span>
        </Link>
    );
};
