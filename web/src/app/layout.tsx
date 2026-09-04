import type { Metadata } from "next";
import { cookies } from "next/headers";
import type { ReactNode } from "react";
import type { Theme } from "../lib/theme";
import { AppProviders } from "../providers/AppProviders";
import "../index.css";

export const metadata: Metadata = {
    title: "WEBYCP",
    description: "Open-source hosting control panel",
};

const RootLayout = async ({ children }: Readonly<{ children: ReactNode }>) => {
    const saved = (await cookies()).get("webycp_theme")?.value;
    const theme: Theme = saved === "dark" ? "dark" : "light";

    return (
        <html lang="en" data-theme={theme} style={{ colorScheme: theme }}>
            <head>
                <meta name="theme-color" content={theme === "dark" ? "#07191f" : "#f2fbf8"} />
            </head>
            <body>
                <AppProviders theme={theme}>{children}</AppProviders>
            </body>
        </html>
    );
};

export default RootLayout;
