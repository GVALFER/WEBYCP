import { Button, Spinner } from "@heroui/react";
import { Server } from "lucide-react";
import { HTTPError } from "reqly-js";
import { useEffect, useState } from "react";
import { Route, Routes } from "react-router-dom";
import useSWR from "swr";

import type { AuthResponse, BootstrapResponse } from "../api/types";
import { AuthScreen } from "../features/auth/AuthScreen";
import { Dashboard } from "../features/dashboard/Dashboard";
import { fetcher, setCsrfToken } from "../lib/api";

const Loading = () => (
  <div className="flex min-h-screen items-center justify-center bg-background text-foreground">
    <div className="flex flex-col items-center gap-4">
      <div className="flex size-12 items-center justify-center rounded-2xl bg-accent text-accent-foreground">
        <Server className="size-6" aria-hidden="true" />
      </div>
      <Spinner aria-label="Loading WEBYCP" />
    </div>
  </div>
);

const Unavailable = ({ retry }: { retry: () => void }) => (
  <div className="flex min-h-screen items-center justify-center bg-background px-6 text-foreground">
    <div className="max-w-md rounded-2xl border border-divider bg-surface p-8 text-center">
      <h1 className="text-xl font-semibold">WEBYCP is unavailable</h1>
      <div className="mt-2 text-sm leading-6 text-foreground-500">
        The control plane could not be reached. Check the server process and try
        again.
      </div>
      <Button className="mt-6" variant="secondary" onPress={retry}>
        Try again
      </Button>
    </div>
  </div>
);

const Root = () => {
  const [sessionChecked, setSessionChecked] = useState(false);
  const {
    data: bootstrap,
    error: bootstrapError,
    isLoading: bootstrapLoading,
    mutate: mutateBootstrap,
  } = useSWR<BootstrapResponse>("bootstrap", fetcher);

  const {
    data: session,
    error: sessionError,
    mutate: mutateSession,
  } = useSWR<AuthResponse>(
    bootstrap && !bootstrap.required ? "auth/me" : null,
    fetcher,
    {
      onError: () => setSessionChecked(true),
      onSuccess: () => setSessionChecked(true),
    },
  );

  useEffect(() => {
    setCsrfToken(session?.csrfToken);
  }, [session?.csrfToken]);

  if (bootstrapLoading && !bootstrap && !bootstrapError) {
    return <Loading />;
  }

  if (bootstrapError || !bootstrap) {
    return <Unavailable retry={() => void mutateBootstrap()} />;
  }

  const authenticated = (next: AuthResponse) => {
    setCsrfToken(next.csrfToken);
    void mutateBootstrap({ required: false }, false);
    void mutateSession(next, false);
  };

  if (bootstrap.required) {
    return <AuthScreen mode="bootstrap" onSuccess={authenticated} />;
  }

  if (!sessionChecked && !session && !sessionError) {
    return <Loading />;
  }

  if (!session) {
    if (
      sessionError instanceof HTTPError &&
      sessionError.response.status !== 401
    ) {
      return <Unavailable retry={() => void mutateSession()} />;
    }
    return <AuthScreen mode="login" onSuccess={authenticated} />;
  }

  return (
    <Dashboard
      session={session}
      onLogout={() => {
        setCsrfToken();
        void mutateSession(undefined, false);
      }}
    />
  );
};

export const App = () => (
  <Routes>
    <Route path="*" element={<Root />} />
  </Routes>
);
