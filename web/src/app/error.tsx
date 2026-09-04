"use client";

import { Button } from "@heroui/react";
import { useEffect } from "react";

type Props = {
    error: Error & { digest?: string };
    reset: () => void;
};

const ErrorPage = ({ error, reset }: Props) => {
    useEffect(() => {
        console.error(error);
    }, [error]);

    return (
        <div className="flex min-h-screen items-center justify-center bg-background px-6 text-foreground">
            <div className="max-w-md rounded-2xl border border-divider bg-surface p-8 text-center">
                <h1 className="text-xl font-semibold">WEBYCP is unavailable</h1>
                <div className="mt-2 text-sm leading-6 text-foreground-500">
                    The control plane could not be reached. Check the server process and try again.
                </div>
                <Button className="mt-6" variant="secondary" onPress={reset}>
                    Try again
                </Button>
            </div>
        </div>
    );
};

export default ErrorPage;
