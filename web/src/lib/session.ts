import "server-only";
import { HTTPError } from "reqly-js";
import * as v from "valibot";
import type { AuthResponse } from "../contracts/types";
import { api } from "./api";

const sessionSchema = v.object({
    user: v.object({
        id: v.string(),
        username: v.string(),
        email: v.string(),
        name: v.string(),
        role: v.picklist(["admin", "user"]),
        mustChangePassword: v.boolean(),
        createdAt: v.string(),
    }),
    csrfToken: v.string(),
    expiresAt: v.string(),
    timezone: v.string(),
});

export const getSession = async (): Promise<AuthResponse | null> => {
    try {
        const body = await api.get("auth/me").json<unknown>();
        return v.parse(sessionSchema, body);
    } catch (error) {
        if (error instanceof HTTPError && error.response.status === 401) {
            return null;
        }
        throw error;
    }
};
