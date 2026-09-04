import { Button } from "@heroui/react";
import { LogOut, Menu, Moon, Server, Sun, X } from "lucide-react";
import { type ReactNode, useState } from "react";

import type { AuthResponse } from "../../api/types";
import { useTheme } from "../../lib/theme";
import { cn } from "../../utils/classnames";
import { nav, page, type View } from "./nav";

type Props = {
  children: ReactNode;
  session: AuthResponse;
  view: View;
  onView: (view: View) => void;
  onLogout: () => void;
};

export const AppShell = ({
  children,
  session,
  view,
  onView,
  onLogout,
}: Props) => {
  const [open, setOpen] = useState(false);
  const { theme, toggleTheme } = useTheme();
  const ThemeIcon = theme === "dark" ? Sun : Moon;

  const select = (next: View) => {
    onView(next);
    setOpen(false);
  };

  return (
    <div className="min-h-screen bg-background text-foreground">
      {open && (
        <button
          className="fixed inset-0 z-40 bg-black/55 backdrop-blur-sm lg:hidden"
          type="button"
          aria-label="Close navigation"
          onClick={() => setOpen(false)}
        />
      )}

      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-50 flex w-64 flex-col border-r border-divider bg-surface/95 p-4 shadow-2xl backdrop-blur-xl transition-transform duration-200 lg:translate-x-0 lg:shadow-none",
          open ? "translate-x-0" : "-translate-x-full",
        )}
      >
        <div className="flex h-14 items-center justify-between px-2">
          <div className="flex items-center gap-3">
            <div className="flex size-9 items-center justify-center rounded-xl bg-accent text-accent-foreground shadow-lg shadow-accent/20">
              <Server className="size-4" aria-hidden="true" />
            </div>
            <div>
              <div className="font-semibold tracking-tight">WEBYCP</div>
              <div className="text-[10px] font-medium tracking-[0.16em] text-foreground-400 uppercase">
                Control panel
              </div>
            </div>
          </div>
          <Button
            className="lg:hidden"
            isIconOnly
            size="sm"
            variant="tertiary"
            aria-label="Close navigation"
            onPress={() => setOpen(false)}
          >
            <X className="size-4" />
          </Button>
        </div>

        <nav className="mt-6 flex-1 space-y-6 overflow-y-auto px-1">
          {nav.map((group) => (
            <div key={group.label}>
              <div className="mb-2 px-3 text-[10px] font-semibold tracking-[0.16em] text-foreground-400 uppercase">
                {group.label}
              </div>
              <div className="space-y-1">
                {group.items.map((item) => {
                  const Icon = item.icon;
                  const active = item.id === view;
                  return (
                    <button
                      key={item.id}
                      type="button"
                      className={cn(
                        "group flex h-10 w-full items-center gap-3 rounded-xl px-3 text-sm font-medium transition",
                        active
                          ? "bg-accent/12 text-accent"
                          : "text-foreground-500 hover:bg-surface-secondary hover:text-foreground",
                      )}
                      onClick={() => select(item.id)}
                    >
                      <Icon
                        className={cn(
                          "size-4 transition",
                          active
                            ? "text-accent"
                            : "text-foreground-400 group-hover:text-foreground",
                        )}
                        aria-hidden="true"
                      />
                      {item.label}
                    </button>
                  );
                })}
              </div>
            </div>
          ))}
        </nav>

        <div className="mt-4 rounded-2xl border border-divider bg-surface-secondary/60 p-3">
          <div className="flex min-w-0 items-center gap-3">
            <div className="flex size-9 shrink-0 items-center justify-center rounded-full bg-accent/15 text-sm font-semibold text-accent">
              {session.user.name.charAt(0).toUpperCase()}
            </div>
            <div className="min-w-0 flex-1">
              <div className="truncate text-sm font-medium">
                {session.user.name}
              </div>
              <div className="truncate text-xs text-foreground-400">
                {session.user.email}
              </div>
            </div>
          </div>
          <div className="mt-3 flex gap-2 border-t border-divider pt-3">
            <Button
              fullWidth
              size="sm"
              variant="tertiary"
              aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} theme`}
              onPress={toggleTheme}
            >
              <ThemeIcon className="size-4" />
              {theme === "dark" ? "Light theme" : "Dark theme"}
            </Button>
            <Button
              isIconOnly
              size="sm"
              variant="tertiary"
              aria-label="Sign out"
              onPress={onLogout}
            >
              <LogOut className="size-4" />
            </Button>
          </div>
        </div>
      </aside>

      <div className="lg:pl-64">
        <header className="sticky top-0 z-30 border-b border-divider bg-background/80 backdrop-blur-xl">
          <div className="flex h-18 items-center gap-4 px-5 sm:px-8 lg:px-10">
            <Button
              className="lg:hidden"
              isIconOnly
              size="sm"
              variant="tertiary"
              aria-label="Open navigation"
              onPress={() => setOpen(true)}
            >
              <Menu className="size-5" />
            </Button>
            <div className="min-w-0">
              <h1 className="truncate text-lg font-semibold tracking-tight sm:text-xl">
                {page[view].title}
              </h1>
              <div className="hidden truncate text-xs text-foreground-400 sm:block">
                {page[view].description}
              </div>
            </div>
          </div>
        </header>

        <main className="mx-auto max-w-[100rem] px-5 py-6 sm:px-8 sm:py-8 lg:px-10">
          {children}
        </main>
      </div>
    </div>
  );
};
