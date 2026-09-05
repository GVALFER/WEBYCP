import type { JobDetail } from "@/contracts/types";
import { api } from "@/lib/api";
import Job from "./job";

const JobPage = async ({ params }: { params: Promise<{ id: string }> }) => {
    const { id } = await params;
    const detail = await api.get(`jobs/${encodeURIComponent(id)}`).json<JobDetail>();
    return <Job detail={detail} />;
};

export default JobPage;
