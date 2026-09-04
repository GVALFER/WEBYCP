"use client";

import { Button } from "@heroui/react";
import { Moon, Server, Sun } from "lucide-react";
import { useTheme } from "@/lib/theme";
import { useSession } from "@/providers/SessionProvider";
import { ProfileForm } from "./actions/profileForm";

export const Setup = () => {
    const session = useSession();
    const { theme, toggleTheme } = useTheme();

    const ThemeIcon = theme === "dark" ? Sun : Moon;

    return (
        <div className="flex min-h-screen items-center justify-center bg-background px-5 py-10 text-foreground">
            <Button
                className="fixed top-5 right-5"
                isIconOnly
                variant="tertiary"
                aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} theme`}
                onPress={toggleTheme}
            >
                <ThemeIcon className="size-4" />
            </Button>
            <section className="panel-card w-full max-w-2xl overflow-hidden">
                <div className="border-b border-divider px-6 py-6 sm:px-8">
                    <div className="flex items-center gap-3">
                        <div className="flex size-10 items-center justify-center rounded-xl bg-accent text-accent-foreground">
                            <Server className="size-5" />
                        </div>
                        <div>
                            <div className="font-semibold">Complete administrator setup</div>
                            <div className="text-xs text-foreground-400">
                                Replace the temporary credentials before managing the server.
                            </div>
                        </div>
                    </div>
                </div>
                <div className="px-6 py-6 sm:px-8">
                    <ProfileForm forced session={session} />
                </div>
            </section>
        </div>
    );
};
