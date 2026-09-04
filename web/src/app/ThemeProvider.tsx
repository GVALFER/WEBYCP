import { type ReactNode, useEffect, useState } from "react";

import {
  readTheme,
  saveTheme,
  ThemeContext,
  type Theme,
} from "../lib/theme";

export const ThemeProvider = ({ children }: { children: ReactNode }) => {
  const [theme, setTheme] = useState<Theme>(readTheme);

  useEffect(() => {
    saveTheme(theme);
  }, [theme]);

  return (
    <ThemeContext
      value={{
        theme,
        toggleTheme: () =>
          setTheme((current) => (current === "dark" ? "light" : "dark")),
      }}
    >
      {children}
    </ThemeContext>
  );
};
