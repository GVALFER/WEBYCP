"use client";

import { Button, Drawer, useOverlayState } from "@heroui/react";
import { Menu } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { ThemeToggle } from "@/components/actions/themeToggle";
import { useSession } from "@/providers/SessionProvider";
import { getPage, Sidebar } from "./sidebar";

export const Header = () => {
    const current = getPage(usePathname());
    const session = useSession();
    const drawer = useOverlayState();

    return (
        <header className="sticky top-0 z-30 border-b border-divider bg-background/78 backdrop-blur-xl">
            <div className="mx-auto flex h-20 w-full max-w-[100rem] items-center gap-4 px-5 sm:px-8 lg:px-10">
                <Drawer state={drawer}>
                    <Button
                        className="lg:hidden"
                        isIconOnly
                        size="sm"
                        variant="tertiary"
                        aria-label="Open navigation"
                        onPress={drawer.open}
                    >
                        <Menu className="size-5" aria-hidden="true" />
                    </Button>
                    <Drawer.Backdrop variant="blur">
                        <Drawer.Content placement="left">
                            <Drawer.Dialog aria-label="Navigation" className="w-[17rem] p-0">
                                <Drawer.CloseTrigger aria-label="Close navigation" />
                                <Drawer.Body className="p-0">
                                    <Sidebar onNavigate={drawer.close} />
                                </Drawer.Body>
                            </Drawer.Dialog>
                        </Drawer.Content>
                    </Drawer.Backdrop>
                </Drawer>

                <div className="min-w-0 flex-1">
                    <h1 className="truncate text-xl font-semibold tracking-[-0.03em]">
                        {current.title}
                    </h1>
                    <div className="mt-0.5 hidden truncate text-xs text-foreground-400 sm:block">
                        {current.description}
                    </div>
                </div>

                <div className="flex items-center gap-2">
                    <ThemeToggle />
                    <div className="mx-1 hidden h-6 w-px bg-divider sm:block" />
                    <Link
                        href="/profile"
                        prefetch={false}
                        aria-label="Open administrator profile"
                        className="hidden items-center gap-2 rounded-xl px-2 py-1.5 text-sm outline-none transition-colors hover:bg-surface-secondary focus-visible:ring-2 focus-visible:ring-focus sm:flex"
                    >
                        <span className="flex size-7 items-center justify-center rounded-lg bg-accent/12 text-xs font-semibold text-accent">
                            {session.user.name.charAt(0).toUpperCase()}
                        </span>
                        <span className="max-w-32 truncate font-medium">{session.user.name}</span>
                    </Link>
                </div>
            </div>
        </header>
    );
};
