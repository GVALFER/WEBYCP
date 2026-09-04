import { redirect } from "next/navigation";
import type { ReactNode } from "react";
import { Content } from "@/components/layout/content";
import { Footer } from "@/components/layout/footer";
import { Header } from "@/components/layout/header";
import { Sidebar } from "@/components/layout/sidebar";
import { Setup } from "@/components/Setup";
import { getSession } from "@/lib/session";
import { SessionProvider } from "@/providers/SessionProvider";

export const dynamic = "force-dynamic";

const ProtectedLayout = async ({ children }: Readonly<{ children: ReactNode }>) => {
    const session = await getSession();

    if (!session) {
        return redirect("/login");
    }

    return (
        <SessionProvider value={session}>
            {session.user.mustChangePassword ? (
                <Setup />
            ) : (
                <div className="min-h-screen bg-background text-foreground">
                    <aside className="fixed inset-y-0 left-0 hidden w-64 border-r border-divider lg:block">
                        <Sidebar />
                    </aside>

                    <div className="flex min-h-screen flex-col lg:pl-64">
                        <Header />
                        <Content className="flex-1">{children}</Content>
                        <Footer />
                    </div>
                </div>
            )}
        </SessionProvider>
    );
};

export default ProtectedLayout;
