"use client";

import { Button } from "@heroui/react";
import { Moon, Sun } from "lucide-react";
import { useTheme } from "@/lib/theme";
import { cn } from "@/utils/classnames";

type Props = {
    className?: string;
};

export const ThemeToggle = ({ className }: Props) => {
    const { theme, toggleTheme } = useTheme();
    const Icon = theme === "dark" ? Sun : Moon;

    return (
        <Button
            className={cn("shrink-0", className)}
            isIconOnly
            size="sm"
            variant="tertiary"
            aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} theme`}
            onPress={toggleTheme}
        >
            <Icon className="size-4" aria-hidden="true" />
        </Button>
    );
};
