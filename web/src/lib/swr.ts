import type { SWRConfiguration } from "swr";
import { fetcher } from "./api";

export const swrConfig: SWRConfiguration = {
    fetcher,
    revalidateOnFocus: true,
    revalidateOnReconnect: false,
    revalidateOnMount: false,
    revalidateIfStale: true,
    shouldRetryOnError: false,
    keepPreviousData: true,
};
