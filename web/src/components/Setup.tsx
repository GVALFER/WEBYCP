"use client";

import { ShieldCheck } from "lucide-react";
import { ThemeToggle } from "@/components/actions/themeToggle";
import { Brand } from "@/components/brand";
import { useSession } from "@/providers/SessionProvider";
import { ProfileForm } from "./actions/profileForm";

export const Setup = () => {
    const session = useSession();

    return (
        <div className="auth-shell flex min-h-screen flex-col text-foreground">
            <header className="flex items-center justify-between px-5 py-5 sm:px-8 sm:py-7">
                <Brand />
                <ThemeToggle />
            </header>
            <main className="flex flex-1 items-center justify-center px-5 py-10 sm:px-8">
                <section className="auth-card w-full max-w-2xl overflow-hidden">
                    <div className="border-b border-divider px-6 py-6 sm:px-8">
                    <div className="flex items-center gap-3">
                        <div className="flex size-10 items-center justify-center rounded-xl border border-accent/20 bg-accent/10 text-accent">
                            <ShieldCheck className="size-5" aria-hidden="true" />
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
            </main>
        </div>
    );
};
