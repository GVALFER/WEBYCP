"use client";

import { createContext, type ReactNode, useContext, useEffect } from "react";

import type { AuthResponse } from "../contracts/types";
import { setCsrfToken } from "../lib/api";

type Props = {
    children: ReactNode;
    value: AuthResponse;
};

const SessionContext = createContext<AuthResponse | null>(null);

export const useSession = () => {
    const session = useContext(SessionContext);
    if (!session) {
        throw new Error("useSession must be used inside SessionProvider");
    }
    return session;
};

export const SessionProvider = ({ children, value }: Props) => {
    useEffect(() => {
        setCsrfToken(value.csrfToken);
        return () => setCsrfToken();
    }, [value.csrfToken]);

    return <SessionContext value={value}>{children}</SessionContext>;
};
