"use client";

import { Button, Spinner } from "@heroui/react";
import { LogOut } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { api, setCsrfToken } from "@/lib/api";

const Logout = () => {
    const router = useRouter();
    const [pending, setPending] = useState(false);

    const logout = async () => {
        setPending(true);

        try {
            await api.post("auth/logout");
        } finally {
            setCsrfToken();
            router.replace("/login");
            router.refresh();
        }
    };

    return (
        <Button
            isIconOnly
            size="sm"
            variant="tertiary"
            aria-label="Sign out"
            isPending={pending}
            onPress={() => void logout()}
        >
            {pending ? (
                <Spinner color="current" size="sm" />
            ) : (
                <LogOut className="size-4" />
            )}
        </Button>
    );
};

export default Logout;
