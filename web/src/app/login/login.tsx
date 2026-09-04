"use client";

import { Button } from "@heroui/react";
import { Check, Moon, Server, Sun } from "lucide-react";
import { useTheme } from "@/lib/theme";
import LoginForm from "./actions/loginForm";

const Login = () => {
    const { theme, toggleTheme } = useTheme();

    const ThemeIcon = theme === "dark" ? Sun : Moon;

    return (
        <div className="auth-shell grid min-h-screen text-foreground lg:grid-cols-[minmax(0,1.15fr)_minmax(28rem,0.85fr)]">
            <Button
                className="fixed top-5 right-5 z-20"
                isIconOnly
                variant="tertiary"
                aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} theme`}
                onPress={toggleTheme}
            >
                <ThemeIcon className="size-4" />
            </Button>
            <div className="relative hidden overflow-hidden border-r border-divider px-12 py-10 lg:flex lg:flex-col lg:justify-between xl:px-18 xl:py-14">
                <div className="auth-glow" />
                <div className="relative flex items-center gap-3 text-lg font-semibold">
                    <div className="flex size-10 items-center justify-center rounded-xl bg-accent text-accent-foreground shadow-lg shadow-accent/20">
                        <Server className="size-5" aria-hidden="true" />
                    </div>
                    <div>
                        <div className="tracking-tight">WEBYCP</div>
                        <div className="text-[10px] font-medium tracking-[0.18em] text-foreground-400 uppercase">
                            Control panel
                        </div>
                    </div>
                </div>

                <div className="relative max-w-2xl">
                    <div className="inline-flex items-center rounded-full border border-accent/20 bg-accent/8 px-3 py-1.5 text-xs font-medium text-accent">
                        Self-hosted by design
                    </div>
                    <h1 className="mt-6 max-w-xl text-4xl font-semibold tracking-[-0.04em] xl:text-5xl">
                        Infrastructure control without the clutter.
                    </h1>
                    <div className="mt-5 max-w-xl text-base leading-7 text-foreground-500 xl:text-lg">
                        One focused control plane for accounts, domains, databases, certificates,
                        automation and backups.
                    </div>
                    <div className="mt-8 flex flex-wrap gap-x-6 gap-y-3 text-sm text-foreground-500">
                        {["Open source", "Ubuntu 24.04", "Agent-driven"].map((item) => (
                            <div key={item} className="flex items-center gap-2">
                                <span className="flex size-5 items-center justify-center rounded-full bg-success/12 text-success">
                                    <Check className="size-3" />
                                </span>
                                {item}
                            </div>
                        ))}
                    </div>
                </div>

                <div className="relative text-xs text-foreground-400">
                    Your server. Your data. Your rules.
                </div>
            </div>

            <div className="flex items-center justify-center px-5 py-10 sm:px-8 lg:bg-surface/35">
                <div className="w-full max-w-104">
                    <div className="mb-10 flex items-center gap-3 lg:hidden">
                        <div className="flex size-10 items-center justify-center rounded-xl bg-accent text-accent-foreground">
                            <Server className="size-5" aria-hidden="true" />
                        </div>
                        <div className="font-semibold">WEBYCP</div>
                    </div>

                    <div className="mb-8">
                        <div className="mb-3 text-xs font-semibold tracking-[0.16em] text-accent uppercase">
                            Control plane
                        </div>
                        <h2 className="text-3xl font-semibold tracking-[-0.035em]">Welcome back</h2>
                        <div className="mt-2 text-sm leading-6 text-foreground-500">
                            Sign in with the credentials shown during installation.
                        </div>
                    </div>

                    <LoginForm />
                </div>
            </div>
        </div>
    );
};

export default Login;
