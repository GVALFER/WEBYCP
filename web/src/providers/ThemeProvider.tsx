"use client";

import { type ReactNode, useEffect, useState } from "react";
import { saveTheme, ThemeContext, type Theme } from "../lib/theme";

type Props = {
    children: ReactNode;
    initial: Theme;
};

export const ThemeProvider = ({ children, initial }: Props) => {
    const [theme, setTheme] = useState(initial);

    useEffect(() => {
        saveTheme(theme);
    }, [theme]);

    return (
        <ThemeContext
            value={{
                theme,
                toggleTheme: () => setTheme((current) => (current === "dark" ? "light" : "dark")),
            }}
        >
            {children}
        </ThemeContext>
    );
};
