import { redirect } from "next/navigation";
import { getSession } from "@/lib/session";
import Profile from "./profile";

const ProfilePage = async () => {
    const session = await getSession();

    if (!session) {
        return redirect("/login");
    }

    return <Profile session={session} />;
};

export default ProfilePage;
