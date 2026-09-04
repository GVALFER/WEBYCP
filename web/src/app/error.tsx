"use client";

import { Button } from "@heroui/react";
import { CircleAlert } from "lucide-react";
import { useEffect } from "react";
import { Brand } from "@/components/brand";

type Props = {
    error: Error & { digest?: string };
    reset: () => void;
};

const ErrorPage = ({ error, reset }: Props) => {
    useEffect(() => {
        console.error(error);
    }, [error]);

    return (
        <div className="auth-shell flex min-h-screen flex-col px-5 py-5 text-foreground sm:px-8 sm:py-7">
            <Brand />
            <main className="flex flex-1 items-center justify-center py-10">
                <section className="auth-card w-full max-w-md p-8 text-center">
                    <div className="mx-auto mb-5 flex size-11 items-center justify-center rounded-xl bg-danger/10 text-danger">
                        <CircleAlert className="size-5" aria-hidden="true" />
                    </div>
                <h1 className="text-xl font-semibold">WEBYCP is unavailable</h1>
                <div className="mt-2 text-sm leading-6 text-foreground-500">
                    The control plane could not be reached. Check the server process and try again.
                </div>
                <Button className="mt-6" variant="secondary" onPress={reset}>
                    Try again
                </Button>
                </section>
            </main>
        </div>
    );
};

export default ErrorPage;
