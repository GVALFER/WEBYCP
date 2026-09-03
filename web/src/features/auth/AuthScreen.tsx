import { Button, Input, Label, TextField } from "@heroui/react";
import { LockKeyhole, Server } from "lucide-react";
import { type FormEvent, useState } from "react";

import { createBootstrap, login, type AuthResponse } from "../../lib/api";
import { errorMessage } from "../../utils/errors";

type Props = {
  mode: "bootstrap" | "login";
  onSuccess: (session: AuthResponse) => void;
};

export const AuthScreen = ({ mode, onSuccess }: Props) => {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const isBootstrap = mode === "bootstrap";

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setPending(true);
    setError("");
    const data = new FormData(event.currentTarget);

    try {
      const credentials = {
        email: String(data.get("email")),
        password: String(data.get("password")),
      };
      const session = isBootstrap
        ? await createBootstrap({
            ...credentials,
            name: String(data.get("name")),
          })
        : await login(credentials);
      onSuccess(session);
    } catch (requestError) {
      setError(await errorMessage(requestError));
    } finally {
      setPending(false);
    }
  };

  return (
    <div className="grid min-h-screen bg-background text-foreground lg:grid-cols-[1.1fr_0.9fr]">
      <div className="hidden border-r border-divider bg-surface/40 p-12 lg:flex lg:flex-col lg:justify-between">
        <div className="flex items-center gap-3 text-lg font-semibold">
          <div className="flex size-10 items-center justify-center rounded-xl bg-accent text-accent-foreground">
            <Server className="size-5" aria-hidden="true" />
          </div>
          WEBYCP
        </div>
        <div className="max-w-xl space-y-5">
          <div className="text-sm font-medium tracking-[0.2em] text-accent uppercase">
            Self-hosted by design
          </div>
          <h1 className="text-5xl font-semibold tracking-tight">
            Your hosting stack, under your control.
          </h1>
          <div className="text-lg leading-8 text-foreground-500">
            A focused control plane for users, domains, databases, SSL, cron
            jobs and backups.
          </div>
        </div>
        <div className="text-sm text-foreground-400">
          Open-source · Ubuntu 24.04
        </div>
      </div>

      <div className="flex items-center justify-center px-6 py-12">
        <div className="w-full max-w-md">
          <div className="mb-8 flex size-12 items-center justify-center rounded-2xl border border-divider bg-surface lg:hidden">
            <Server className="size-6 text-accent" aria-hidden="true" />
          </div>
          <div className="mb-8 space-y-2">
            <h2 className="text-3xl font-semibold tracking-tight">
              {isBootstrap ? "Create your administrator" : "Welcome back"}
            </h2>
            <div className="text-foreground-500">
              {isBootstrap
                ? "Set up the first local account for this server."
                : "Sign in to manage this WEBYCP server."}
            </div>
          </div>

          <form className="space-y-5" onSubmit={submit}>
            {isBootstrap && (
              <TextField name="name" isRequired fullWidth>
                <Label>Full name</Label>
                <Input autoComplete="name" minLength={2} maxLength={80} />
              </TextField>
            )}
            <TextField name="email" type="email" isRequired fullWidth>
              <Label>Email address</Label>
              <Input autoComplete="email" />
            </TextField>
            <TextField name="password" type="password" isRequired fullWidth>
              <Label>Password</Label>
              <Input
                autoComplete={isBootstrap ? "new-password" : "current-password"}
                minLength={isBootstrap ? 12 : undefined}
                maxLength={128}
              />
              {isBootstrap && (
                <div className="mt-2 text-xs text-foreground-400">
                  Use at least 12 characters.
                </div>
              )}
            </TextField>

            {error && (
              <div className="rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">
                {error}
              </div>
            )}

            <Button
              type="submit"
              variant="primary"
              fullWidth
              isDisabled={pending}
            >
              <LockKeyhole className="size-4" aria-hidden="true" />
              {pending
                ? "Please wait…"
                : isBootstrap
                  ? "Create administrator"
                  : "Sign in"}
            </Button>
          </form>
        </div>
      </div>
    </div>
  );
};
