import type { SWRConfiguration } from "swr";
import { fetcher } from "./api";

export const swrConfig: SWRConfiguration = {
    fetcher,
    revalidateOnFocus: true,
    revalidateOnReconnect: false,
    revalidateIfStale: false,
    shouldRetryOnError: false,
    keepPreviousData: true,
};
