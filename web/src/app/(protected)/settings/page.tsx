import type { CertificateListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import Settings from "./settings";

const SettingsPage = async () => {
    const certificates = await api
        .get("certificates", { searchParams: { kind: "panel", page: 1, size: 10 } })
        .json<CertificateListResponse>();

    return <Settings certificates={certificates} />;
};

export default SettingsPage;
