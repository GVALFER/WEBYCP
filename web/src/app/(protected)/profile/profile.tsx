"use client";

import type { AuthResponse } from "@/contracts/types";
import { ProfileForm } from "@/components/actions/profileForm";
import useSWR from "swr";

type ProfileProps = {
    session: AuthResponse;
};

const Profile = ({ session }: ProfileProps) => {
    const { data } = useSWR<AuthResponse>("auth/me", {
        fallbackData: session,
    });

    return (
        <section className="panel-card max-w-3xl overflow-hidden">
            <div className="border-b border-divider px-6 py-5">
                <h2 className="text-base font-semibold">Administrator profile</h2>
                <div className="mt-1 text-sm text-foreground-500">
                    Update the login name, contact email, timezone or password.
                </div>
            </div>
            <div className="px-6 py-6">
                <ProfileForm session={data ?? session} />
            </div>
        </section>
    );
};

export default Profile;
