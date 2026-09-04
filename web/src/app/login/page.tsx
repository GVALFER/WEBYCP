import { redirect } from "next/navigation";
import { getSession } from "@/lib/session";
import Login from "./login";

export const dynamic = "force-dynamic";

const LoginPage = async () => {
    const session = await getSession();

    if (session) {
        return redirect("/");
    }

    return <Login />;
};

export default LoginPage;
