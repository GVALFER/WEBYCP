"use client";

import { LockKeyhole } from "lucide-react";
import { ThemeToggle } from "@/components/actions/themeToggle";
import { Brand } from "@/components/brand";
import LoginForm from "./actions/loginForm";

const Login = () => (
    <div className="auth-shell flex min-h-screen flex-col text-foreground">
        <header className="flex items-center justify-between px-5 py-5 sm:px-8 sm:py-7">
            <Brand />
            <ThemeToggle />
        </header>

        <main className="flex flex-1 items-center justify-center px-5 py-10 sm:px-8">
            <section
                className="auth-card w-full max-w-108 p-6 sm:p-8"
                aria-labelledby="login-title"
            >
                <div className="mb-7 flex size-11 items-center justify-center rounded-xl border border-accent/20 bg-accent/10 text-accent">
                    <LockKeyhole className="size-5" strokeWidth={1.8} aria-hidden="true" />
                </div>

                <div className="mb-7">
                    <div className="mb-2 text-[10px] font-semibold tracking-[0.18em] text-accent uppercase">
                        Secure access
                    </div>
                    <h1 id="login-title" className="text-2xl font-semibold tracking-[-0.035em]">
                        Welcome back
                    </h1>
                    <div className="mt-2 text-sm leading-6 text-foreground-500">
                        Sign in with the administrator credentials created during installation.
                    </div>
                </div>

                <LoginForm />

                <div className="mt-7 border-t border-divider pt-5 text-center text-[11px] text-foreground-400">
                    Self-hosted infrastructure control
                </div>
            </section>
        </main>

        <footer className="px-5 py-5 text-center text-[10px] tracking-[0.14em] text-foreground-400 uppercase">
            Your server · Your data · Your control
        </footer>
    </div>
);

export default Login;
