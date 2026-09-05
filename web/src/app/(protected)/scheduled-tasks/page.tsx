import type {
    AccountListResponse,
    ScheduledTaskListResponse,
    ServiceSettings,
} from "@/contracts/types";
import { api } from "@/lib/api";
import { getPageQuery, syncPage, type PageProps } from "@/components/table/paginationServer";
import Tasks from "./tasks";

const TasksPage = async ({ searchParams }: PageProps) => {
    const query = await getPageQuery("/scheduled-tasks", searchParams);

    const [accounts, tasks, settings] = await Promise.all([
        api.get("accounts", { searchParams: { page: 1, size: 100 } }).json<AccountListResponse>(),
        api.get("scheduled-tasks", { searchParams: query }).json<ScheduledTaskListResponse>(),
        api.get("service-settings").json<ServiceSettings>(),
    ]);

    await syncPage("/scheduled-tasks", searchParams, query, tasks.pagination);

    return <Tasks accounts={accounts} tasks={tasks} settings={settings} />;
};

export default TasksPage;
