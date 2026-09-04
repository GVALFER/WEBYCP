"use client";

import { Toast } from "@heroui/react";
import type { ReactNode } from "react";
import { SWRConfig } from "swr";
import { swrConfig } from "../lib/swr";
import type { Theme } from "../lib/theme";
import { ThemeProvider } from "./ThemeProvider";

type Props = {
    children: ReactNode;
    theme: Theme;
};

export const AppProviders = ({ children, theme }: Props) => (
    <ThemeProvider initial={theme}>
        <SWRConfig value={swrConfig}>{children}</SWRConfig>
        <Toast.Provider placement="bottom end" />
    </ThemeProvider>
);
