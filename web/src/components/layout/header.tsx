"use client";

import { Button, Drawer } from "@heroui/react";
import { Menu } from "lucide-react";
import { usePathname } from "next/navigation";
import { getPage, Sidebar } from "./sidebar";

export const Header = () => {
    const current = getPage(usePathname());

    return (
        <header className="sticky top-0 z-30 border-b border-divider bg-background/80 backdrop-blur-xl">
            <div className="flex h-18 items-center gap-4 px-5 sm:px-8 lg:px-10">
                <Drawer>
                    <Button
                        className="lg:hidden"
                        isIconOnly
                        size="sm"
                        variant="tertiary"
                        aria-label="Open navigation"
                    >
                        <Menu className="size-5" />
                    </Button>
                    <Drawer.Backdrop variant="blur">
                        <Drawer.Content placement="left">
                            <Drawer.Dialog aria-label="Navigation" className="p-0">
                                <Drawer.CloseTrigger aria-label="Close navigation" />
                                <Drawer.Body className="p-0">
                                    <Sidebar />
                                </Drawer.Body>
                            </Drawer.Dialog>
                        </Drawer.Content>
                    </Drawer.Backdrop>
                </Drawer>

                <div className="min-w-0">
                    <h1 className="truncate text-lg font-semibold tracking-tight sm:text-xl">
                        {current.title}
                    </h1>
                    <div className="hidden truncate text-xs text-foreground-400 sm:block">
                        {current.description}
                    </div>
                </div>
            </div>
        </header>
    );
};
