"use client";

import { Button } from "@heroui/react";
import { LogOut } from "lucide-react";
import { useRouter } from "next/navigation";
import { api, setCsrfToken } from "@/lib/api";

const Logout = () => {
    const router = useRouter();

    const logout = async () => {
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
            onPress={() => void logout()}
        >
            <LogOut className="size-4" />
        </Button>
    );
};

export default Logout;
