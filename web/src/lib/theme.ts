import { createContext, useContext } from "react";

export type Theme = "light" | "dark";

type ThemeState = {
    theme: Theme;
    toggleTheme: () => void;
};

const cookieName = "webycp_theme";

export const ThemeContext = createContext<ThemeState | null>(null);

export const applyTheme = (theme: Theme) => {
    document.documentElement.dataset.theme = theme;
    document.documentElement.style.colorScheme = theme;
    document
        .querySelector('meta[name="theme-color"]')
        ?.setAttribute("content", theme === "dark" ? "#07191f" : "#f2fbf8");
};

export const saveTheme = (theme: Theme) => {
    document.cookie = `${cookieName}=${theme}; Path=/; Max-Age=31536000; SameSite=Lax`;
    applyTheme(theme);
};

export const useTheme = () => {
    const value = useContext(ThemeContext);
    if (!value) throw new Error("useTheme must be used inside ThemeProvider");
    return value;
};
